package variants

import (
	"fmt"

	"github.com/serverledge-faas/serverledge/internal/function"
)

func MergeAndValidate(existing []function.Variant, loaded []function.Variant) ([]function.Variant, error) {
	// Mappa per evitare duplicati ID
	seen := map[string]function.Variant{}

	// helper: aggiunge e controlla
	add := func(v function.Variant) error {
		if v.ID == "" {
			return fmt.Errorf("variant id is empty")
		}
		if v.Runtime == "" {
			return fmt.Errorf("variant %s: runtime is empty", v.ID)
		}
		if v.EntryPoint == "" {
			return fmt.Errorf("variant %s: entry_point is empty", v.ID)
		}
		// energia: non forzo >0 (può essere 0 in prototipo), ma almeno presente
		// output: non forzo sempre, dipende dal tuo schema

		if _, ok := seen[v.ID]; ok {
			return fmt.Errorf("duplicate variant id: %s", v.ID)
		}
		seen[v.ID] = v
		return nil
	}

	for _, v := range existing {
		if err := add(v); err != nil {
			return nil, err
		}
	}
	for _, v := range loaded {
		if err := add(v); err != nil {
			return nil, err
		}
	}

	// preserva ordine: existing prima, poi loaded
	merged := make([]function.Variant, 0, len(seen))
	for _, v := range existing {
		merged = append(merged, seen[v.ID])
		delete(seen, v.ID)
	}
	for _, v := range loaded {
		if vv, ok := seen[v.ID]; ok {
			merged = append(merged, vv)
			delete(seen, v.ID)
		}
	}

	return merged, nil
}
