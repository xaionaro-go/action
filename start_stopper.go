package ctxflow

import (
	"context"

	"github.com/xaionaro-go/observability"
	"github.com/xaionaro-go/xsync"
)

type StartStopperBackend interface {
	Start(context.Context, ...any) error
	Stop(context.Context) error
}

type StartStopperBackendFuncs struct {
	StartFunc func(context.Context, ...any) error
	StopFunc  func(context.Context) error
}

func (s StartStopperBackendFuncs) Start(ctx context.Context, args ...any) error {
	return s.StartFunc(ctx, args...)
}

func (s StartStopperBackendFuncs) Stop(ctx context.Context) error {
	return s.StopFunc(ctx)
}

var _ StartStopperBackend = StartStopperBackendFuncs{}

type StartStopper[T StartStopperBackend] struct {
	StartStopper   T
	CancelFunc     context.CancelFunc
	Locker         xsync.RWMutex
	StopResultChan chan error
}

func (a *StartStopper[T]) Start(
	ctx context.Context,
	args ...any,
) error {
	return xsync.DoA2R1(ctx, &a.Locker, a.StartLocked, ctx, args)
}

func (a *StartStopper[T]) StartLocked(
	ctx context.Context,
	args []any,
) error {
	if a.CancelFunc != nil {
		return ErrAlreadyRunning{}
	}
	ctx, cancelFn := context.WithCancel(ctx)
	a.CancelFunc = cancelFn
	err := a.StartStopper.Start(ctx, args...)
	if err != nil {
		return err
	}
	a.StopResultChan = make(chan error, 1)
	observability.Go(ctx, func(ctx context.Context) {
		<-ctx.Done()
		xsync.RDoA1(ctx, &a.Locker, a.doStopLocked, ctx)
	})
	return nil
}

func (a *StartStopper[T]) doStopLocked(
	ctx context.Context,
) {
	a.StopResultChan <- a.StartStopper.Stop(ctx)
	close(a.StopResultChan)
	a.CancelFunc = nil
}

func (a *StartStopper[T]) Stop() error {
	return xsync.RDoR1(context.Background(), &a.Locker, a.StopLocked)
}

func (a *StartStopper[T]) StopLocked() error {
	cancelFn := a.CancelFunc
	if cancelFn == nil {
		return ErrAlreadyNotRunning{}
	}
	cancelFn()
	return <-a.StopResultChan
}

func (a *StartStopper[T]) IsRunning() bool {
	return xsync.RDoR1(context.Background(), &a.Locker, a.IsRunningLocked)
}

func (a *StartStopper[T]) IsRunningLocked() bool {
	return a.CancelFunc != nil
}
