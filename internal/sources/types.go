package sources

// FileMeta — метаданные файла из Drive.
type FileMeta struct {
	ID           string // Drive file ID
	Name         string
	MimeType     string
	Size         int64
	ModifiedTime string // RFC3339
	Path         string // относительный путь (папки + имя файла)
	Parents      []string
}
