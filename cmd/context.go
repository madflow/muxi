package cmd

import "context"

type startOptionsKey struct{}

func withStartOptions(ctx context.Context, opts startCommandOptions) context.Context {
	return context.WithValue(ctx, startOptionsKey{}, opts)
}

func startOptionsFromContext(ctx context.Context) startCommandOptions {
	if value, ok := ctx.Value(startOptionsKey{}).(startCommandOptions); ok {
		return value
	}
	return startCommandOptions{}
}
