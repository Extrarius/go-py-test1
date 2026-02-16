package pipelines

import (
	"fmt"
	"strings"

	"github.com/Extrarius/go-py-test1/internal/report"
)

// ReportFrom собирает отчёт из результата Run() и метаданных файлов.
// themes и keyFacts можно заполнить отдельно (например, из LLM); здесь themes собирается из саммари папок.
func ReportFrom(global *GlobalSummary, docSummaries []DocSummary, folderSummaries []FolderSummary, meta report.Metadata) *report.Report {
	toc := make([]report.TOCEntry, 0, len(docSummaries))
	for _, d := range docSummaries {
		toc = append(toc, report.TOCEntry{Path: d.Path, Description: d.Summary})
	}
	var themes strings.Builder
	for _, f := range folderSummaries {
		label := f.FolderPath
		if label == "" {
			label = "(корень)"
		}
		line := f.Summary
		if idx := strings.IndexAny(line, "\n"); idx > 0 {
			line = line[:idx]
		}
		if len(line) > 150 {
			line = line[:147] + "..."
		}
		themes.WriteString(fmt.Sprintf("- **%s**: %s\n", label, line))
	}
	execSummary := ""
	if global != nil {
		execSummary = global.ExecutiveSummary
	}
	return report.Build(execSummary, strings.TrimSpace(themes.String()), "", toc, meta)
}
