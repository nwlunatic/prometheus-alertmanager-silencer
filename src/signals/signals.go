package signals

import (
	"context"
	"os"
	"os/signal"
	"syscall"
)

type stopper interface {
	Stop(ctx context.Context) error
}

func BindGracefulStop(ctx context.Context, stoppers ...stopper) <-chan error {
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, syscall.SIGINT, syscall.SIGTERM)
	errChan := make(chan error)
	go func() {
		defer close(errChan)
		<-signalChan
		for _, s := range stoppers {
			err := s.Stop(ctx)
			if err != nil {
				errChan <- err
			}
		}
	}()

	return errChan
}
