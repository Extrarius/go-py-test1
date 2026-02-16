package sources

import (
	"context"
	"os"
	"path/filepath"
	"strings"
)

// LocalSource — источник файлов из локальной папки (без скачивания).
type LocalSource struct {
	rootDir string
}

// NewLocalSource создаёт источник по локальному пути rootDir.
func NewLocalSource(rootDir string) (*LocalSource, error) {
	abs, err := filepath.Abs(rootDir)
	if err != nil {
		return nil, err
	}
	if err := os.MkdirAll(abs, 0755); err != nil {
		return nil, err
	}
	return &LocalSource{rootDir: abs}, nil
}

// ListRecursive обходит rootDir рекурсивно и возвращает метаданные файлов.
func (l *LocalSource) ListRecursive(ctx context.Context) ([]FileMeta, error) {
	var out []FileMeta
	err := filepath.Walk(l.rootDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		rel, _ := filepath.Rel(l.rootDir, path)
		rel = filepath.ToSlash(rel)
		if strings.HasPrefix(rel, "..") {
			return nil
		}
		out = append(out, FileMeta{
			ID:           path,
			Name:         info.Name(),
			MimeType:     "",
			Size:         info.Size(),
			ModifiedTime: "",
			Path:         rel,
		})
		return nil
	})
	return out, err
}

// DownloadToCache для локального источника возвращает путь к файлу без изменений.
func (l *LocalSource) DownloadToCache(ctx context.Context, meta FileMeta) (string, error) {
	localPath := filepath.Join(l.rootDir, filepath.FromSlash(meta.Path))
	if _, err := os.Stat(localPath); err != nil {
		return "", err
	}
	return localPath, nil
}
