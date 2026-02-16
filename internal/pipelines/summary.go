package pipelines

import (
	"context"
	"fmt"
	"log"
	"path/filepath"
	"strings"
	"time"

	"github.com/Extrarius/go-py-test1/internal/llm"
	"github.com/Extrarius/go-py-test1/internal/llm/prompts"
	"github.com/Extrarius/go-py-test1/internal/storage"
)

// LLMCaller вызывается для получения ответа (для подстановки клиента и кеша).
type LLMCaller interface {
	Chat(ctx context.Context, messages []llm.Message) (string, error)
	Model() string
}

// Run выполняет иерархическую саммаризацию: chunks → doc → folder → global.
// failedDocs — пути документов, по которым не удалось получить саммари (логируются, выполнение продолжается).
func Run(ctx context.Context, docInputs []DocInput, caller LLMCaller, cache *storage.LLMCache) (global *GlobalSummary, docSummaries []DocSummary, folderSummaries []FolderSummary, failedDocs []string, err error) {
	model := caller.Model()
	docSummaries, failedDocs, err = chunksToDocs(ctx, docInputs, caller, cache, model)
	if err != nil {
		return nil, nil, nil, failedDocs, err
	}
	if len(docSummaries) == 0 {
		return nil, nil, nil, failedDocs, nil
	}
	folderSummaries, err = docsToFolders(ctx, docSummaries, caller, cache, model)
	if err != nil {
		return nil, docSummaries, nil, failedDocs, err
	}
	global, err = foldersToGlobal(ctx, folderSummaries, caller, cache, model)
	if err != nil {
		return nil, docSummaries, folderSummaries, failedDocs, err
	}
	return global, docSummaries, folderSummaries, failedDocs, nil
}

func chunksToDocs(ctx context.Context, docInputs []DocInput, caller LLMCaller, cache *storage.LLMCache, model string) ([]DocSummary, []string, error) {
	var out []DocSummary
	var failed []string
	total := len(docInputs)
	for idx, doc := range docInputs {
		if len(doc.Chunks) == 0 {
			continue
		}
		if total > 1 {
			log.Printf("summarize [%d/%d] %s", idx+1, total, doc.Path)
		}
		var sb strings.Builder
		for i, c := range doc.Chunks {
			if i > 0 {
				sb.WriteString("\n\n")
			}
			sb.WriteString(fmt.Sprintf("--- Чанк %d ---\n%s", c.ChunkIndex+1, c.Text))
		}
		inputText := sb.String()
		prompt := fmt.Sprintf(prompts.SummarizeChunks, inputText)
		key := storage.Key(prompts.SummarizeChunks, inputText, model)
		if cached, ok := cache.Get(key); ok {
			out = append(out, DocSummary{FileID: doc.FileID, Path: doc.Path, Summary: cached})
			continue
		}
		msg, err := caller.Chat(ctx, []llm.Message{{Role: "user", Content: prompt}})
		if err != nil {
			log.Printf("ошибка саммари %q: %v", doc.Path, err)
			failed = append(failed, doc.Path)
			continue
		}
		_ = cache.Set(key, msg)
		out = append(out, DocSummary{FileID: doc.FileID, Path: doc.Path, Summary: msg})
		select {
		case <-time.After(2 * time.Second):
		case <-ctx.Done():
			return out, failed, ctx.Err()
		}
	}
	return out, failed, nil
}

func pathDir(path string) string {
	if path == "" {
		return ""
	}
	path = filepath.ToSlash(path)
	i := strings.LastIndex(path, "/")
	if i <= 0 {
		return ""
	}
	return path[:i]
}

func docsToFolders(ctx context.Context, docSummaries []DocSummary, caller LLMCaller, cache *storage.LLMCache, model string) ([]FolderSummary, error) {
	byFolder := make(map[string][]DocSummary)
	for _, d := range docSummaries {
		dir := pathDir(d.Path)
		byFolder[dir] = append(byFolder[dir], d)
	}
	var out []FolderSummary
	for folderPath, docs := range byFolder {
		var sb strings.Builder
		for _, d := range docs {
			sb.WriteString(fmt.Sprintf("Документ: %s\n%s\n\n", d.Path, d.Summary))
		}
		inputText := strings.TrimSpace(sb.String())
		prompt := fmt.Sprintf(prompts.SummarizeDocs, inputText)
		key := storage.Key(prompts.SummarizeDocs, inputText, model)
		if cached, ok := cache.Get(key); ok {
			out = append(out, FolderSummary{FolderPath: folderPath, Summary: cached})
			continue
		}
		msg, err := caller.Chat(ctx, []llm.Message{{Role: "user", Content: prompt}})
		if err != nil {
			return nil, fmt.Errorf("folder %q: %w", folderPath, err)
		}
		_ = cache.Set(key, msg)
		out = append(out, FolderSummary{FolderPath: folderPath, Summary: msg})
		log.Printf("summarized folder: %s", folderPath)
		select {
		case <-time.After(2 * time.Second):
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}
	return out, nil
}

func foldersToGlobal(ctx context.Context, folderSummaries []FolderSummary, caller LLMCaller, cache *storage.LLMCache, model string) (*GlobalSummary, error) {
	var sb strings.Builder
	for _, f := range folderSummaries {
		label := f.FolderPath
		if label == "" {
			label = "(корень)"
		}
		sb.WriteString(fmt.Sprintf("--- %s ---\n%s\n\n", label, f.Summary))
	}
	inputText := strings.TrimSpace(sb.String())
	prompt := fmt.Sprintf(prompts.SummarizeFolder, inputText)
	key := storage.Key(prompts.SummarizeFolder, inputText, model)
	if cached, ok := cache.Get(key); ok {
		return &GlobalSummary{ExecutiveSummary: cached}, nil
	}
	msg, err := caller.Chat(ctx, []llm.Message{{Role: "user", Content: prompt}})
	if err != nil {
		return nil, fmt.Errorf("global: %w", err)
	}
	_ = cache.Set(key, msg)
	log.Printf("global summary done")
	return &GlobalSummary{ExecutiveSummary: msg}, nil
}
