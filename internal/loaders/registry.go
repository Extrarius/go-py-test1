// Регистр loaders по расширению: легко добавить новый формат через Register(".ext", loader).
package loaders

import (
	"path/filepath"
	"strings"
	"sync"
)

var (
	registry   = make(map[string]Loader)
	registryMu sync.RWMutex
)

func init() {
	Register(".pdf", LoaderFunc(loadPDF))
	Register(".docx", LoaderFunc(loadDOCX))
	Register(".doc", LoaderFunc(loadDOCX))
	Register(".txt", LoaderFunc(loadText))
	Register(".md", LoaderFunc(loadText))
	Register(".markdown", LoaderFunc(loadText))
	Register(".log", LoaderFunc(loadText))
	Register(".csv", LoaderFunc(loadText))
}

// Register добавляет загрузчик для расширения (с точкой, например ".pdf").
func Register(ext string, l Loader) {
	registryMu.Lock()
	defer registryMu.Unlock()
	ext = strings.ToLower(ext)
	if ext != "" && ext[0] != '.' {
		ext = "." + ext
	}
	registry[ext] = l
}

// GetLoader возвращает загрузчик по пути к файлу (по расширению). Если формата нет в регистре — nil, false.
func GetLoader(path string) (Loader, bool) {
	ext := strings.ToLower(filepath.Ext(path))
	registryMu.RLock()
	defer registryMu.RUnlock()
	l, ok := registry[ext]
	return l, ok
}

// LoadFile загружает файл, если для расширения есть загрузчик; иначе возвращает "", false, nil.
func LoadFile(path string) (text string, ok bool, err error) {
	l, ok := GetLoader(path)
	if !ok {
		return "", false, nil
	}
	text, err = l.Load(path)
	if err != nil {
		return "", true, err
	}
	return text, true, nil
}
