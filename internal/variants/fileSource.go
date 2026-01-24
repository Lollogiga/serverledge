package variants

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/serverledge-faas/serverledge/internal/function"
	"github.com/serverledge-faas/serverledge/utils"
)

type FileSource struct {
	BaseDir string
}

func (f *FileSource) Type() string {
	return "file"
}

func (f *FileSource) Exists(profile string) bool {
	if profile == "" {
		return false
	}

	profileDir := filepath.Join(f.BaseDir, profile)
	jsonPath := filepath.Join(profileDir, profile+".json")

	// Check directory
	if stat, err := os.Stat(profileDir); err != nil || !stat.IsDir() {
		return false
	}

	// Check JSON file
	if stat, err := os.Stat(jsonPath); err != nil || stat.IsDir() {
		return false
	}

	return true
}

func (f *FileSource) Load(_ context.Context, fn *function.Function) ([]function.Variant, error) {
	if fn == nil {
		return nil, fmt.Errorf("nil function")
	}
	if fn.Name == "" {
		return nil, fmt.Errorf("function name is empty")
	}

	logical := fn.Name
	profileDir := filepath.Join(f.BaseDir, logical)
	jsonPath := filepath.Join(profileDir, logical+".json")

	log.Printf("[variants] loading variants from %s\n", jsonPath)

	data, err := os.ReadFile(jsonPath)
	if err != nil {
		return nil, fmt.Errorf("cannot read variants file %s: %w", jsonPath, err)
	}

	var wrapper struct {
		Variants []function.Variant `json:"variants"`
	}
	if err := json.Unmarshal(data, &wrapper); err != nil {
		return nil, fmt.Errorf("invalid variants json %s: %w", jsonPath, err)
	}
	if len(wrapper.Variants) == 0 {
		return nil, fmt.Errorf("no variants defined in %s", jsonPath)
	}

	// Prepare code for each variant
	for i := range wrapper.Variants {
		v := &wrapper.Variants[i]

		if v.ID == "" {
			return nil, fmt.Errorf("variant with empty id in %s", jsonPath)
		}
		if v.Src == "" {
			return nil, fmt.Errorf("variant %s has empty src (json=%s)", v.ID, jsonPath)
		}

		// Src is relative to variants/<logical_name>/
		srcPath := filepath.Join(profileDir, v.Src)

		log.Printf("[variants] preparing variant %s (src=%s)\n", v.ID, srcPath)

		tarBytes, err := utils.ReadSourcesAsTar(srcPath)
		if err != nil {
			return nil, fmt.Errorf("failed creating tar for variant %s (src=%s): %w", v.ID, srcPath, err)
		}

		v.TarCode = base64.StdEncoding.EncodeToString(tarBytes)
	}

	return wrapper.Variants, nil
}
