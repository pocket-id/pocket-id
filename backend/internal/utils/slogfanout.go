package utils

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"slices"
)

// This file contains code adapted from https://github.com/samber/slog-multi
// Source: https://github.com/samber/slog-multi/blob/ced84707f45ec9848138349ed58de178eedaa6f2/pipe.go
// Copyright (C) 2023 Samuel Berthe
// License: MIT (https://github.com/samber/slog-multi/blob/ced84707f45ec9848138349ed58de178eedaa6f2/LICENSE)

// LogFanoutHandler is a slog.Handler that sends logs to multiple destinations
type LogFanoutHandler []slog.Handler

// Implements slog.Handler
func (h LogFanoutHandler) Enabled(ctx context.Context, l slog.Level) bool {
	for i := range h {
		if h[i].Enabled(ctx, l) {
			return true
		}
	}

	return false
}

// Implements slog.Handler
func (h LogFanoutHandler) Handle(ctx context.Context, r slog.Record) error {
	errs := make([]error, 0)
	for i := range h {
		if h[i].Enabled(ctx, r.Level) {
			err := try(func() error {
				return h[i].Handle(ctx, r.Clone())
			})
			if err != nil {
				errs = append(errs, err)
			}
		}
	}

	return errors.Join(errs...)
}

// Implements slog.Handler
func (h LogFanoutHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	res := make(LogFanoutHandler, len(h))
	for i, v := range h {
		res[i] = v.WithAttrs(slices.Clone(attrs))
	}
	return res
}

// Implements slog.Handler
func (h LogFanoutHandler) WithGroup(name string) slog.Handler {
	// https://cs.opensource.google/go/x/exp/+/46b07846:slog/handler.go;l=247
	if name == "" {
		return h
	}

	res := make(LogFanoutHandler, len(h))
	for i, v := range h {
		res[i] = v.WithGroup(name)
	}
	return res
}

func try(callback func() error) (err error) {
	defer func() {
		r := recover()
		if r != nil {
			if e, ok := r.(error); ok {
				err = e
			} else {
				err = fmt.Errorf("unexpected error: %+v", r)
			}
		}
	}()

	err = callback()

	return
}
