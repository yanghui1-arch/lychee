package ocr

import (
	"crypto/sha256"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/yanghui1-arch/lychee/models"
)

// Unpack embedded, verified weights locally; inference never downloads models.
func prepareModels(cache string) (string, error) {
	files := make(map[string][]byte)
	digest := sha256.New()
	err := fs.WalkDir(models.OCR, "ocr", func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			return nil
		}
		data, err := models.OCR.ReadFile(path)
		if err != nil {
			return err
		}
		files[path] = data
		digest.Write([]byte(path))
		digest.Write(data)
		return nil
	})
	if err != nil {
		return "", err
	}
	root := filepath.Join(cache, "Lychee", fmt.Sprintf("pp-ocrv6-small-%x", digest.Sum(nil)[:8]))
	for path, data := range files {
		dest := filepath.Join(root, filepath.FromSlash(path))
		existing, err := os.ReadFile(dest)
		if err == nil && sha256.Sum256(existing) == sha256.Sum256(data) {
			continue
		}
		if err := os.MkdirAll(filepath.Dir(dest), 0700); err != nil {
			return "", err
		}
		file, err := os.CreateTemp(filepath.Dir(dest), "model-*")
		if err != nil {
			return "", err
		}
		name := file.Name()
		_, err = file.Write(data)
		closeErr := file.Close()
		if err == nil {
			err = closeErr
		}
		if err == nil {
			err = os.Rename(name, dest)
		}
		os.Remove(name)
		if err != nil {
			return "", err
		}
	}
	return filepath.Join(root, "ocr"), nil
}
