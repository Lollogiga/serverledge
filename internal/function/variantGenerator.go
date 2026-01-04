package function

import (
	"encoding/json"
	"fmt"
	"os"
)

func LoadVariantsFromFile(fn *Function) ([]Variant, error) {
	// TODO: path hardcoded solo per ora
	path := "variants.json"

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("cannot read variants file: %w", err)
	}

	var variants []Variant
	if err := json.Unmarshal(data, &variants); err != nil {
		return nil, fmt.Errorf("invalid variants file: %w", err)
	}

	return variants, nil
}
