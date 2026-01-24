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

	// 1. Caso deterministico: profilo su file
	if f.FileSource != nil && f.FileSource.Exists(fn.Name) {
		return f.FileSource, nil
	}

	// 2. Caso future-proof: generazione automatica
	if fn.AllowApprox && f.GeneratorSource != nil {
		return f.GeneratorSource, nil
	}

	// 3. Nessuna sorgente disponibile
	return nil, errors.New("no variant source available")
}
