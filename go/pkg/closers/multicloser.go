package closers

import (
	"io"
	"log"
	"os"
	"os/signal"
	"sync"
	"sync/atomic"
	"syscall"
)

type multicloser []io.Closer

func (s multicloser) Close() (err error) {
	for i := len(s) - 1; i >= 0; i-- {
		cl := s[i]
		log.Printf("Closing %#T\n", cl)
		if err1 := cl.Close(); err == nil && err1 != nil {
			log.Println("Closer err", err1)
			err = err1
		}
		log.Printf("Closing %#T Ok\n", cl)
	}
	return
}

type closers struct {
	mu sync.Mutex

	closers multicloser
	closed  uint32
}

func New() *closers {
	return &closers{}
}

func (c *closers) AddCloser(cl io.Closer) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.closers = append(c.closers, cl)
}

func (c *closers) Close() (err error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if atomic.CompareAndSwapUint32(&c.closed, 0, 1) {
		err = c.closers.Close()
	}

	return
}

// handle Ctrl-C or kill -INT
func (c *closers) ExitOnSigInt() *closers {
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt)
	signal.Notify(sig, syscall.SIGHUP)
	go func() {
		code := <-sig
		log.Printf("Handle signal, will exit")

		if c != nil {
			log.Println("Calling closers")
			err := c.Close()
			if err != nil {
				log.Printf("err calling Close handler: %v", err)
			}
		} else {
			log.Println("no closers")
		}

		if code == syscall.SIGHUP {
			os.Exit(111)
		} else {
			os.Exit(1)
		}
	}()
	return c
}

/*
type closer struct {
	f func() error
}

func (c closer) Close() error {
	return c.f()
}

func Closer(f func() error) io.Closer {
	return closer{f}
}*/
