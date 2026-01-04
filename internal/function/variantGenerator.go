package function

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/serverledge-faas/serverledge/internal/variant"
)

func LoadVariantsFromFile(fn *Function) ([]variant.Variant, error) {
	// TODO: path hardcoded solo per ora
	path := "variants.json"

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("cannot read variants file: %w", err)
	}

	var variants []variant.Variant
	if err := json.Unmarshal(data, &variants); err != nil {
		return nil, fmt.Errorf("invalid variants file: %w", err)
	}

	return variants, nil
}
