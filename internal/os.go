package internal

import (
	"context"
	"log"
	"os"
	"os/signal"
	"sync"
	"syscall"
)

var (
	ctx    context.Context
	cancel context.CancelFunc
	once   sync.Once
)

func initSig() {
	var (
		sigC chan os.Signal
	)
	ctx, cancel = context.WithCancel(context.Background())

	sigC = make(chan os.Signal)
	signal.Notify(sigC, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		defer cancel()
		sig := <-sigC
		log.Printf("received signal %s\n", sig)
		signal.Reset()
	}()
}

// GetGracefulCtx returns a context which is cancelled when the process receives a signal
func GetGracefulCtx() context.Context {
	once.Do(initSig)
	return ctx
}

// CancelContext manually cancels the context
func CancelContext() {
	cancel()
}
