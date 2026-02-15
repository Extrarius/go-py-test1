package storage

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

// LLMCache кеширует ответы LLM по ключу hash(prompt + text + model).
type LLMCache struct {
	dir string
	mu  sync.RWMutex
}

// NewLLMCache создаёт кеш в dir (например .cache/llm).
func NewLLMCache(dir string) (*LLMCache, error) {
	dir = filepath.Join(dir, "llm")
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("mkdir llm cache: %w", err)
	}
	return &LLMCache{dir: dir}, nil
}

// Key возвращает ключ кеша по prompt, inputText и model.
func Key(prompt, inputText, model string) string {
	h := sha256.Sum256([]byte(prompt + "\x00" + inputText + "\x00" + model))
	return hex.EncodeToString(h[:])
}

// Get возвращает закешированный ответ, если есть.
func (c *LLMCache) Get(key string) (string, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	p := filepath.Join(c.dir, key+".txt")
	b, err := os.ReadFile(p)
	if err != nil {
		return "", false
	}
	return string(b), true
}

// Set сохраняет ответ в кеш.
func (c *LLMCache) Set(key, value string) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	p := filepath.Join(c.dir, key+".txt")
	return os.WriteFile(p, []byte(value), 0644)
}
