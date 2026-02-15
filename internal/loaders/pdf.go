package loaders

import (
	"bytes"
	"fmt"
	"io"

	"github.com/ledongthuc/pdf"
)

func loadPDF(path string) (string, error) {
	f, r, err := pdf.Open(path)
	if err != nil {
		return "", fmt.Errorf("pdf open: %w", err)
	}
	defer f.Close()
	b, err := r.GetPlainText()
	if err != nil {
		return "", fmt.Errorf("pdf text: %w", err)
	}
	var buf bytes.Buffer
	if _, err := io.Copy(&buf, b); err != nil {
		return "", fmt.Errorf("pdf read: %w", err)
	}
	return Normalize(buf.String()), nil
}
