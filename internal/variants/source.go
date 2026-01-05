package variants

import (
	"context"

	"github.com/serverledge-faas/serverledge/internal/function"
)

type Source interface {
	Type() string
	Load(ctx context.Context, fn *function.Function) ([]function.Variant, error)
}
