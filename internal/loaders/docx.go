package loaders

import (
	"fmt"

	"github.com/lu4p/cat"
)

func loadDOCX(path string) (string, error) {
	txt, err := cat.File(path)
	if err != nil {
		return "", fmt.Errorf("docx: %w", err)
	}
	return Normalize(txt), nil
}
