package report

// Build собирает отчёт из готовых секций. themes и keyFacts могут быть пустыми.
func Build(execSummary, themes, keyFacts string, toc []TOCEntry, meta Metadata) *Report {
	if toc == nil {
		toc = []TOCEntry{}
	}
	if meta.TypeCounts == nil {
		meta.TypeCounts = make(map[string]int)
	}
	return &Report{
		ExecutiveSummary: execSummary,
		Themes:           themes,
		KeyFacts:         keyFacts,
		TableOfContents:  toc,
		Metadata:         meta,
	}
}
