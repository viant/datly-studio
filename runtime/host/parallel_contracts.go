package host

import (
	"context"

	"golang.org/x/sync/errgroup"
)

// compileContracts keeps each compile result in definition order. The caller
// gives every worker its own type catalog; no generation state is shared here.
func compileContracts[T any](ctx context.Context, definitions []definition, compile func(context.Context, definition) (T, error)) ([]T, error) {
	results := make([]T, len(definitions))
	group, scoped := errgroup.WithContext(ctx)
	group.SetLimit(4)
	for i := range definitions {
		i := i
		group.Go(func() error {
			result, err := compile(scoped, definitions[i])
			if err == nil {
				results[i] = result
			}
			return err
		})
	}
	if err := group.Wait(); err != nil {
		return nil, err
	}
	return results, nil
}
