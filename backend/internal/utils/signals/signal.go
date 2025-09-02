package signals

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
)

/*
This code is adapted from:
https://github.com/kubernetes-sigs/controller-runtime/blob/8499b67e316a03b260c73f92d0380de8cd2e97a1/pkg/manager/signals/signal.go
Copyright 2017 The Kubernetes Authors.
License: Apache2 (https://github.com/kubernetes-sigs/controller-runtime/blob/8499b67e316a03b260c73f92d0380de8cd2e97a1/LICENSE)
*/

var onlyOneSignalHandler = make(chan struct{})

// SignalContext returns a context that is canceled when the application receives an interrupt signal.
// A second signal forces an immediate shutdown.
func SignalContext(parentCtx context.Context) context.Context {
	close(onlyOneSignalHandler) // Panics when called twice

	ctx, cancel := context.WithCancel(parentCtx)

	sigCh := make(chan os.Signal, 2)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-sigCh
		slog.Info("Received interrupt signal. Shutting down…")
		cancel()

		<-sigCh
		slog.Warn("Received a second interrupt signal. Forcing an immediate shutdown.")
		os.Exit(1)
	}()

	return ctx
}
