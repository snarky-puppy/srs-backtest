package internal

import (
	"context"
	"os"
	"os/signal"
	"syscall"
)

// create a function that returns  a context.Context that will be cancelled when program receives signal
// eg: SIGINT (ctrl+c) or SIGTERM (docker stop)
func Graceful() context.Context {
	ctx, cancel := context.WithCancel(context.Background())

	// Create a channel to receive OS signals
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)

	// Start a goroutine to receive OS signal
	go func() {
		<-sig
		cancel()
	}()

	return ctx
}

func Round4(value float64) float64 {
	// round value to 4 decimals
	return float64(int(value*10000)) / 10000
}

func Round2(value float64) float64 {
	// round value to 2 decimals
	return float64(int(value*100)) / 100
}

type Closer interface {
	Close() error
}

func Close(closer Closer) {
	if closer != nil {
		_ = closer.Close()
	}
}
