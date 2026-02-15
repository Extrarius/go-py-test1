package loaders

import (
	"fmt"
	"os"
)

var utf8BOM = []byte{0xEF, 0xBB, 0xBF}

func loadText(path string) (string, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("read: %w", err)
	}
	s := string(raw)
	if len(raw) >= 3 && raw[0] == utf8BOM[0] && raw[1] == utf8BOM[1] && raw[2] == utf8BOM[2] {
		s = string(raw[3:])
	}
	return Normalize(s), nil
}
