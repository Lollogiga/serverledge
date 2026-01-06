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

func (f *FileSource) Load(ctx context.Context, fn *function.Function) ([]function.Variant, error) {

	// Resolve profile
	profile := fn.VariantsProfileID
	if profile == "" {
		return nil, fmt.Errorf("variants-profile-id is empty")
	}

	profileDir := filepath.Join(f.BaseDir, profile)
	jsonPath := filepath.Join(profileDir, profile+".json")

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

	for i := range wrapper.Variants {
		v := &wrapper.Variants[i]

		if v.ID == "" {
			return nil, fmt.Errorf("variant with empty id in %s", jsonPath)
		}
		if v.Src == "" {
			return nil, fmt.Errorf("variant %s has empty src", v.ID)
		}

		srcPath := filepath.Join(profileDir, v.Src)

		log.Printf("[variants] preparing variant %s (src=%s)\n", v.ID, srcPath)
		tarBytes, err := utils.ReadSourcesAsTar(srcPath)
		if err != nil {
			return nil, fmt.Errorf(
				"failed creating tar for variant %s (src=%s): %w",
				v.ID, srcPath, err,
			)
		}

		v.TarCode = base64.StdEncoding.EncodeToString(tarBytes)
	}

	return wrapper.Variants, nil
}
