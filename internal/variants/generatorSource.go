package variants

import (
	"context"
	"errors"

	"github.com/serverledge-faas/serverledge/internal/function"
)

type GeneratorSource struct{}

func (g *GeneratorSource) Type() string {
	return "generator"
}

func (g *GeneratorSource) Load(ctx context.Context, fn *function.Function) ([]function.Variant, error) {
	return nil, errors.New("generator source not implemented yet")
}
