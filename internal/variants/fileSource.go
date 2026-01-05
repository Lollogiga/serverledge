package variants

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/serverledge-faas/serverledge/internal/function"
)

type FileSource struct {
	BaseDir string
}

func (f *FileSource) Type() string {
	return "file"
}

func (f *FileSource) Load(ctx context.Context, fn *function.Function) ([]function.Variant, error) {

	profile := fn.VariantsProfileID
	if profile == "" {
		profile = fn.Name
	}

	path := filepath.Join(f.BaseDir, profile+".json")

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var wrapper struct {
		Variants []function.Variant `json:"variants"`
	}

	if err := json.Unmarshal(data, &wrapper); err != nil {
		return nil, err
	}

	return wrapper.Variants, nil
}
