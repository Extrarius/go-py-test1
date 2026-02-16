package storage

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/Extrarius/go-py-test1/internal/report"
)

const summaryResultName = "summary_result.json"

// SaveSummaryResult сохраняет отчёт в dir (для команды report).
func SaveSummaryResult(dir string, r *report.Report) error {
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	path := filepath.Join(dir, summaryResultName)
	raw, err := json.MarshalIndent(r, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, raw, 0644)
}

// LoadSummaryResult загружает отчёт из dir.
func LoadSummaryResult(dir string) (*report.Report, error) {
	path := filepath.Join(dir, summaryResultName)
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("summary result: %w", err)
	}
	var r report.Report
	if err := json.Unmarshal(raw, &r); err != nil {
		return nil, err
	}
	return &r, nil
}
