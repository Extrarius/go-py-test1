package storage

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// CachedFile — запись о файле, сохранённом после ingest (для summarize).
type CachedFile struct {
	FileID    string `json:"file_id"`
	Path      string `json:"path"`
	LocalPath string `json:"local_path"`
	Size      int64  `json:"size"`
}

const fileListName = "file_list.json"

// SaveFileList сохраняет список файлов в dir (например .cache).
func SaveFileList(dir string, list []CachedFile) error {
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	path := filepath.Join(dir, fileListName)
	raw, err := json.MarshalIndent(list, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, raw, 0644)
}

// LoadFileList загружает список файлов из dir.
func LoadFileList(dir string) ([]CachedFile, error) {
	path := filepath.Join(dir, fileListName)
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("file list: %w", err)
	}
	var list []CachedFile
	if err := json.Unmarshal(raw, &list); err != nil {
		return nil, err
	}
	return list, nil
}
