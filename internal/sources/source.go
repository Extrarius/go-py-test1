package sources

import "context"

// Source — интерфейс источника файлов (Drive API, локальная папка, публичная ссылка и т.д.).
type Source interface {
	// ListRecursive возвращает метаданные всех файлов рекурсивно.
	ListRecursive(ctx context.Context) ([]FileMeta, error)
	// DownloadToCache скачивает/подготавливает файл и возвращает локальный путь.
	DownloadToCache(ctx context.Context, meta FileMeta) (localPath string, err error)
}
