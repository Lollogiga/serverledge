package variants

import (
	"errors"

	"github.com/serverledge-faas/serverledge/internal/function"
)

type Factory struct {
	FileSource      Source
	GeneratorSource Source
}

func (f *Factory) GetSource(fn *function.Function) (Source, error) {

	// Caso esplicito: profilo varianti definito
	if fn.VariantsProfileID != "" {
		return f.FileSource, nil
	}

	// Caso future-proof: approx ma senza file
	if fn.AllowApprox && f.GeneratorSource != nil {
		return f.GeneratorSource, nil
	}

	return nil, errors.New("no variant source available")
}
