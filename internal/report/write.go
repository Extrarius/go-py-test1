package report

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// WriteToFile записывает отчёт в файл. format: "md" или "json".
func WriteToFile(r *Report, path, format string) error {
	format = strings.ToLower(strings.TrimSpace(format))
	var data []byte
	switch format {
	case "md", "markdown":
		data = []byte(r.Markdown())
	case "json":
		var err error
		data, err = r.JSON()
		if err != nil {
			return err
		}
	default:
		return fmt.Errorf("report: неизвестный формат %q", format)
	}
	if dir := filepath.Dir(path); dir != "." {
		_ = os.MkdirAll(dir, 0755)
	}
	return os.WriteFile(path, data, 0644)
}
