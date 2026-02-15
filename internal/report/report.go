// Package report — формат результата: Executive Summary, карта тем, оглавление, метаданные; вывод в Markdown и JSON.
package report

import (
	"encoding/json"
	"fmt"
	"strings"
)

// TOCEntry — пункт оглавления (документ и краткое описание).
type TOCEntry struct {
	Path        string `json:"path"`
	Description string `json:"description"`
}

// Metadata — метаданные по папке: кол-во файлов, типы, размеры.
type Metadata struct {
	FileCount  int            `json:"file_count"`
	TypeCounts map[string]int `json:"type_counts"` // расширение или MIME -> количество
	TotalSize  int64          `json:"total_size"`
}

// Report — итоговый отчёт саммаризации.
type Report struct {
	ExecutiveSummary string     `json:"executive_summary"`
	Themes           string     `json:"themes"`     // ключевые темы/разделы
	KeyFacts         string     `json:"key_facts"` // цифры, даты, ссылки
	TableOfContents  []TOCEntry `json:"table_of_contents"`
	Metadata         Metadata   `json:"metadata"`
}

// Markdown возвращает отчёт в формате Markdown.
func (r *Report) Markdown() string {
	var b strings.Builder
	b.WriteString("# Саммари папки\n\n")
	b.WriteString("## Executive Summary\n\n")
	b.WriteString(r.ExecutiveSummary)
	b.WriteString("\n\n")
	if r.Themes != "" {
		b.WriteString("## Карта тем\n\n")
		b.WriteString(r.Themes)
		b.WriteString("\n\n")
	}
	if r.KeyFacts != "" {
		b.WriteString("## Ключевые факты\n\n")
		b.WriteString(r.KeyFacts)
		b.WriteString("\n\n")
	}
	b.WriteString("## Оглавление\n\n")
	for _, e := range r.TableOfContents {
		desc := e.Description
		if len(desc) > 200 {
			desc = desc[:197] + "..."
		}
		b.WriteString(fmt.Sprintf("- **%s** — %s\n", e.Path, desc))
	}
	b.WriteString("\n## Метаданные\n\n")
	b.WriteString(fmt.Sprintf("- Файлов: %d\n", r.Metadata.FileCount))
	b.WriteString(fmt.Sprintf("- Общий размер: %d байт\n", r.Metadata.TotalSize))
	if len(r.Metadata.TypeCounts) > 0 {
		b.WriteString("- По типам:\n")
		for t, n := range r.Metadata.TypeCounts {
			b.WriteString(fmt.Sprintf("  - %s: %d\n", t, n))
		}
	}
	return b.String()
}

// JSON возвращает отчёт в формате JSON (без экранирования HTML).
func (r *Report) JSON() ([]byte, error) {
	return json.MarshalIndent(r, "", "  ")
}
