package pipelines

import "github.com/Extrarius/go-py-test1/internal/chunking"

// DocInput — входные данные по одному документу (чанки уже разбиты).
type DocInput struct {
	FileID string
	Path   string
	Chunks []chunking.Chunk
}

// DocSummary — саммари одного документа.
type DocSummary struct {
	FileID  string
	Path    string
	Summary string
}

// FolderSummary — саммари одной папки (обобщение документов).
type FolderSummary struct {
	FolderPath string
	Summary    string
}

// GlobalSummary — итоговый саммари всей папки/проекта.
type GlobalSummary struct {
	ExecutiveSummary string
}
