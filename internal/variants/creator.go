package variants

import (
	"fmt"
	"log"

	"github.com/serverledge-faas/serverledge/internal/function"
)

// CreateInternalVariants materializes runtime functions for each variant.
// Variants are stored directly in Etcd, bypassing the HTTP API layer.
func CreateInternalVariants(
	logicalFn *function.Function,
	variants []function.Variant,
) error {

	if logicalFn == nil {
		return fmt.Errorf("nil function passed to CreateInternalVariants")
	}
	if logicalFn.Name == "" {
		return fmt.Errorf("logical function name is empty")
	}
	if len(variants) == 0 {
		return fmt.Errorf("no variants provided")
	}

	for i, v := range variants {

		// ✅ BASE VARIANT keeps the logical name EXACTLY
		internalName := ""
		if i == 0 {
			internalName = logicalFn.Name
		} else {
			internalName = fmt.Sprintf("%s-%s", logicalFn.Name, v.ID)
		}

		// Idempotency check
		if _, exists := function.GetFunction(internalName); exists {
			log.Printf("[variants] function %s already exists, skipping\n", internalName)
			continue
		}

		// Safety check: code must already be prepared
		if v.TarCode == "" {
			return fmt.Errorf("variant %s has empty TarCode (src=%s)", v.ID, v.Src)
		}

		log.Printf("[variants] creating function %s (variant_id=%s)\n", internalName, v.ID)

		variantFn := &function.Function{
			// Identity
			Name:        internalName,
			LogicalName: logicalFn.Name,
			VariantID:   v.ID,

			// Runtime
			Runtime:         v.Runtime,
			Handler:         v.EntryPoint,
			TarFunctionCode: v.TarCode,

			MemoryMB:       logicalFn.MemoryMB,
			CPUDemand:      logicalFn.CPUDemand,
			MaxConcurrency: logicalFn.MaxConcurrency,
			CustomImage:    logicalFn.CustomImage,
			Signature:      logicalFn.Signature,

			// Scheduling metadata
			EnergyProfile: &v.Energy,
			OutputModel:   &v.Output,
		}

		if err := variantFn.SaveToEtcd(); err != nil {
			return fmt.Errorf("failed saving function %s to etcd: %w", internalName, err)
		}
	}

	return nil
}
