package variants

import (
	"context"
	"fmt"
	"log"

	"github.com/serverledge-faas/serverledge/internal/function"
)

// CreateInternalVariants materializes runtime functions for each variant.
// Variants are stored directly in Etcd, bypassing the HTTP API layer.
func CreateInternalVariants(ctx context.Context, fn *function.Function) error {

	if fn == nil {
		return fmt.Errorf("nil function passed to CreateInternalVariants")
	}

	// Only logical functions can spawn variants
	if !fn.AllowApprox {
		return nil
	}

	for _, v := range fn.Variants {

		internalName := fmt.Sprintf("%s-%s", fn.Name, v.ID)

		// Idempotency check
		if _, exists := function.GetFunction(internalName); exists {
			log.Printf("[variants] variant function %s already exists, skipping\n", internalName)
			continue
		}

		// 🔒 Safety check: code must already be prepared
		if v.TarCode == "" {
			return fmt.Errorf(
				"variant %s has empty TarCode (src=%s): FileSource not executed?",
				v.ID, v.Src,
			)
		}

		log.Printf("[variants] creating variant runtime function %s\n", internalName)

		variantFn := &function.Function{
			Name:           internalName,
			Runtime:        v.Runtime,
			MemoryMB:       fn.MemoryMB,
			CPUDemand:      fn.CPUDemand,
			MaxConcurrency: fn.MaxConcurrency,
			Handler:        v.EntryPoint,

			// 🔑 CRITICAL FIX
			TarFunctionCode: v.TarCode,

			CustomImage: fn.CustomImage,
			Signature:   fn.Signature,

			// Semantics
			AllowApprox:       false,
			Variants:          nil,
			VariantsProfileID: fn.VariantsProfileID,
		}

		if err := variantFn.SaveToEtcd(); err != nil {
			return fmt.Errorf(
				"failed saving variant function %s to etcd: %w",
				internalName,
				err,
			)
		}
	}

	return nil
}
