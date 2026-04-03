package nocontextinstruct

import "context"

// Flagged: context.Context stored as a struct field.
type BadService struct {
	ctx  context.Context // want `context\.Context should not be stored in a struct`
	name string
}

// Flagged: context.Context as an embedded field.
type BadEmbedded struct {
	context.Context // want `context\.Context should not be stored in a struct`
}

// Not flagged: context.Context as a function parameter.
type GoodService struct {
	name string
}

func (s *GoodService) Do(ctx context.Context) error {
	_ = ctx

	return nil
}

// Not flagged: struct with no context fields.
type SimpleConfig struct {
	Host string
	Port int
}
