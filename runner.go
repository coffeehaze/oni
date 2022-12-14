// Copyright 2022 coffeehaze. All rights reserved.
// Use of this source code is governed by a MIT style
// license that can be found in the LICENSE file.

package oni

import (
	"context"
	"log"
	"os"
	"os/signal"
	"sync"
	"time"
)

type Runner struct {
	Context   context.Context
	Timeout   time.Duration
	Syscall   []os.Signal
	Consumers []*Consumer
}

func SyscallOpt(syscall ...os.Signal) []os.Signal {
	return syscall
}

func ConsumerOpt(consumers ...*Consumer) []*Consumer {
	return consumers
}

func (r *Runner) Start() {
	for _, consumer := range r.Consumers {
		go consumer.run(r.Context)
	}

	wait := make(chan struct{})
	go func() {
		s := make(chan os.Signal, 1)

		//syscall.SIGINT, syscall.SIGTERM, syscall.SIGHUP
		signal.Notify(s, r.Syscall...)
		<-s
		log.Println("shutting down")

		timeoutFunc := time.AfterFunc(r.Timeout, func() {
			log.Printf("timeout %d ms has been elapsed, force exit", r.Timeout.Milliseconds())
			os.Exit(0)
		})

		defer timeoutFunc.Stop()

		var wg sync.WaitGroup

		for i, consumer := range r.Consumers {
			wg.Add(1)
			sequence := i
			c := consumer
			go func() {
				defer wg.Done()

				log.Printf("cleaning up process %d", sequence)

				if err := c.closeConsumers(); err != nil {
					log.Printf("consumer %d clean up failed: %s", sequence, err.Error())
					return
				}
				if err := c.closeProducers(); err != nil {
					log.Printf("producer %d clean up failed: %s", sequence, err.Error())
					return
				}

				log.Printf("process %d was shutdown gracefully", sequence)
			}()
		}
		wg.Wait()
		close(wait)
	}()

	<-wait
}
