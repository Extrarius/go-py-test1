// Точка входа: саммари папки Google Drive.
// Подкоманды: ingest, summarize, report. Без подкоманды: ingest → summarize → report.
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/Extrarius/go-py-test1/internal/chunking"
	"github.com/Extrarius/go-py-test1/internal/config"
	"github.com/Extrarius/go-py-test1/internal/loaders"
	"github.com/Extrarius/go-py-test1/internal/llm"
	"github.com/Extrarius/go-py-test1/internal/pipelines"
	"github.com/Extrarius/go-py-test1/internal/report"
	"github.com/Extrarius/go-py-test1/internal/sources"
	"github.com/Extrarius/go-py-test1/internal/storage"
)

var (
	flCacheDir  = flag.String("cache-dir", "", "каталог кеша (по умолчанию .cache)")
	flFolderID  = flag.String("folder-id", "", "ID папки Google Drive (переопределяет .env)")
	flVerbose   = flag.Bool("verbose", false, "подробный вывод")
	flMaxFiles  = flag.Int("max-files", 0, "макс. файлов для обработки (0 = без лимита)")
	flForce     = flag.Bool("force", false, "перекачать файлы при ingest")
	flMode      = flag.String("mode", "fast", "режим саммаризации: fast|deep")
	flFormat    = flag.String("format", "md", "формат отчёта: md|json")
	flOutput    = flag.String("output", "summary.md", "файл вывода отчёта")
)

func main() {
	flag.Parse()
	subcmd := strings.ToLower(strings.TrimSpace(flag.Arg(0)))

	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("config: %v", err)
	}
	if *flCacheDir != "" {
		cfg.CacheDir, _ = filepath.Abs(*flCacheDir)
	}
	if *flFolderID != "" {
		cfg.FolderID = *flFolderID
	}

	switch subcmd {
	case "ingest":
		runIngest(cfg)
	case "summarize":
		runSummarize(cfg)
	case "report":
		runReport(cfg)
	case "":
		runIngest(cfg)
		runSummarize(cfg)
		runReport(cfg)
	default:
		log.Fatalf("неизвестная подкоманда %q. Использование: ingest | summarize | report", subcmd)
	}
}

func runIngest(cfg *config.Config) {
	ctx := context.Background()
	src, err := newSource(ctx, cfg)
	if err != nil {
		log.Fatalf("источник: %v", err)
	}
	list, err := src.ListRecursive(ctx)
	if err != nil {
		log.Fatalf("list: %v", err)
	}
	if *flVerbose {
		log.Printf("найдено файлов: %d", len(list))
	}
	if *flMaxFiles > 0 && len(list) > *flMaxFiles {
		list = list[:*flMaxFiles]
	}
	total := len(list)
	var cached []storage.CachedFile
	var skipped []string
	for i, meta := range list {
		if *flVerbose || total <= 20 {
			log.Printf("ingest [%d/%d] %s", i+1, total, meta.Path)
		}
		if *flForce && cfg.Source == "drive" {
			dir := filepath.Join(cfg.CacheDir, "downloads", meta.ID)
			_ = os.RemoveAll(dir)
		}
		localPath, err := src.DownloadToCache(ctx, meta)
		if err != nil {
			log.Printf("пропуск %q: %v", meta.Path, err)
			skipped = append(skipped, meta.Path)
			continue
		}
		cached = append(cached, storage.CachedFile{
			FileID:    meta.ID,
			Path:      meta.Path,
			LocalPath: localPath,
			Size:      meta.Size,
		})
		_, _, _ = loaders.LoadFile(localPath)
	}
	if err := storage.SaveFileList(cfg.CacheDir, cached); err != nil {
		log.Printf("сохранение списка файлов: %v", err)
	}
	log.Printf("ingest завершён: успешно %d, пропущено %d", len(cached), len(skipped))
	if len(skipped) > 0 {
		log.Printf("не обработано при ingest: %v", skipped)
	}
}

// newSource создаёт источник файлов по конфигу (drive или local).
func newSource(ctx context.Context, cfg *config.Config) (sources.Source, error) {
	switch cfg.Source {
	case "local":
		dir := cfg.LocalDir
		if dir == "" {
			dir = cfg.CacheDir
		}
		return sources.NewLocalSource(dir)
	default:
		if cfg.FolderID == "" {
			return nil, fmt.Errorf("GOOGLE_DRIVE_FOLDER_ID не задан (--folder-id или .env)")
		}
		return sources.NewDriveSource(ctx, cfg.CredentialsPath, cfg.FolderID, cfg.CacheDir, cfg.TokenPath)
	}
}

func runSummarize(cfg *config.Config) {
	if cfg.OpenRouterAPIKey == "" {
		log.Fatal("OPENROUTER_API_KEY не задан (.env)")
	}
	list, err := storage.LoadFileList(cfg.CacheDir)
	if err != nil {
		log.Fatalf("summarize: %v (сначала выполните ingest)", err)
	}
	if *flMaxFiles > 0 && len(list) > *flMaxFiles {
		list = list[:*flMaxFiles]
	}
	var docInputs []pipelines.DocInput
	var loadSkipped []string
	opts := chunking.DefaultOptions()
	if cfg.ChunkTokens > 0 {
		opts.MaxTokens = cfg.ChunkTokens
	}
	typeCounts := make(map[string]int)
	var totalSize int64
	for i, f := range list {
		if *flVerbose {
			log.Printf("загрузка [%d/%d] %s", i+1, len(list), f.Path)
		}
		text, ok, err := loaders.LoadFile(f.LocalPath)
		if !ok || err != nil {
			loadSkipped = append(loadSkipped, f.Path)
			if err != nil {
				log.Printf("пропуск %q: %v", f.Path, err)
			}
			continue
		}
		chunks := chunking.Split(text, f.FileID, opts)
		docInputs = append(docInputs, pipelines.DocInput{FileID: f.FileID, Path: f.Path, Chunks: chunks})
		ext := filepath.Ext(f.Path)
		if ext != "" {
			typeCounts[strings.ToLower(ext)]++
		}
		totalSize += f.Size
	}
	if len(loadSkipped) > 0 {
		log.Printf("не загружено (формат/ошибка): %d — %v", len(loadSkipped), loadSkipped)
	}
	if len(docInputs) == 0 {
		log.Fatal("summarize: нет загруженного текста (ingest + поддерживаемые форматы)")
	}
	ctx := context.Background()
	concurrency := cfg.LLMConcurrency
	if concurrency <= 0 {
		concurrency = 1
	}
	client := llm.NewClient(cfg.OpenRouterAPIKey, cfg.OpenRouterModel, 120*time.Second, concurrency)
	cache, err := storage.NewLLMCache(cfg.CacheDir)
	if err != nil {
		log.Fatalf("llm cache: %v", err)
	}
	global, docSums, folderSums, failedDocs, err := pipelines.Run(ctx, docInputs, client, cache)
	if err != nil {
		log.Fatalf("pipeline: %v", err)
	}
	if len(failedDocs) > 0 {
		log.Printf("не обработано при саммаризации: %d — %v", len(failedDocs), failedDocs)
	}
	if global == nil {
		log.Printf("summarize: нет успешных саммари, отчёт не сохранён")
		return
	}
	meta := report.Metadata{
		FileCount:  len(list),
		TypeCounts: typeCounts,
		TotalSize:  totalSize,
	}
	r := pipelines.ReportFrom(global, docSums, folderSums, meta)
	if err := storage.SaveSummaryResult(cfg.CacheDir, r); err != nil {
		log.Printf("сохранение результата: %v", err)
	}
	log.Printf("summarize завершён")
}

func runReport(cfg *config.Config) {
	r, err := storage.LoadSummaryResult(cfg.CacheDir)
	if err != nil {
		log.Fatalf("report: %v (сначала выполните summarize)", err)
	}
	outPath := *flOutput
	if outPath == "" {
		outPath = "summary.md"
	}
	format := strings.ToLower(*flFormat)
	if format != "md" && format != "json" {
		format = "md"
	}
	if err := report.WriteToFile(r, outPath, format); err != nil {
		log.Fatalf("report: %v", err)
	}
	if *flVerbose {
		fmt.Fprintf(os.Stderr, "отчёт записан: %s\n", outPath)
	}
	log.Printf("отчёт: %s", outPath)
}
