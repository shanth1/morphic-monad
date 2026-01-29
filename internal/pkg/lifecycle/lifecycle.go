package lifecycle

import "context"

type WorkerFunc func(ctx context.Context) error

func (f WorkerFunc) Run(ctx context.Context) error {
	return f(ctx)
}
