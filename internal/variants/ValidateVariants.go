package variants

import (
	"fmt"

	"github.com/serverledge-faas/serverledge/internal/function"
)

func ValidateVariantsOnly(variants []function.Variant) error {
	if len(variants) == 0 {
		return fmt.Errorf("no variants provided")
	}

	seen := map[string]bool{}

	for i, v := range variants {
		if v.ID == "" {
			return fmt.Errorf("variant with empty id at index %d", i)
		}
		if seen[v.ID] {
			return fmt.Errorf("duplicate variant id: %s", v.ID)
		}
		seen[v.ID] = true

		if v.Runtime == "" {
			return fmt.Errorf("variant %s has empty runtime", v.ID)
		}
		if v.EntryPoint == "" {
			return fmt.Errorf("variant %s has empty entry point", v.ID)
		}
	}

	return nil
}
