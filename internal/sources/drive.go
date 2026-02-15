package sources

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"google.golang.org/api/drive/v3"
	"google.golang.org/api/googleapi"
	"google.golang.org/api/option"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

const (
	driveScope       = drive.DriveReadonlyScope
	mimeTypeFolder   = "application/vnd.google-apps.folder"
	mimeTypeShortcut = "application/vnd.google-apps.shortcut"
)

// DriveSource — источник файлов из Google Drive.
type DriveSource struct {
	srv      *drive.Service
	folderID string
	cacheDir string
	tokenPath string
}

// NewDriveSource создаёт клиент Drive (OAuth по credentialsPath, token в tokenPath).
func NewDriveSource(ctx context.Context, credentialsPath, folderID, cacheDir, tokenPath string) (*DriveSource, error) {
	b, err := os.ReadFile(credentialsPath)
	if err != nil {
		return nil, fmt.Errorf("credentials: %w", err)
	}
	config, err := google.ConfigFromJSON(b, driveScope)
	if err != nil {
		return nil, fmt.Errorf("oauth config: %w", err)
	}
	client, err := getClient(ctx, config, tokenPath)
	if err != nil {
		return nil, err
	}
	srv, err := drive.NewService(ctx, option.WithHTTPClient(client))
	if err != nil {
		return nil, fmt.Errorf("drive service: %w", err)
	}
	downloadsDir := filepath.Join(cacheDir, "downloads")
	if err := os.MkdirAll(downloadsDir, 0755); err != nil {
		return nil, fmt.Errorf("mkdir cache: %w", err)
	}
	return &DriveSource{
		srv:       srv,
		folderID:  folderID,
		cacheDir:  downloadsDir,
		tokenPath: tokenPath,
	}, nil
}

func getClient(ctx context.Context, config *oauth2.Config, tokenPath string) (*http.Client, error) {
	tok, err := tokenFromFile(tokenPath)
	if err != nil {
		tok, err = getTokenFromWeb(ctx, config)
		if err != nil {
			return nil, err
		}
		if err := saveToken(tokenPath, tok); err != nil {
			log.Printf("warning: save token: %v", err)
		}
	}
	return config.Client(ctx, tok), nil
}

func tokenFromFile(path string) (*oauth2.Token, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	tok := &oauth2.Token{}
	err = json.NewDecoder(f).Decode(tok)
	return tok, err
}

func getTokenFromWeb(ctx context.Context, config *oauth2.Config) (*oauth2.Token, error) {
	authURL := config.AuthCodeURL("state-token", oauth2.AccessTypeOffline, oauth2.SetAuthURLParam("prompt", "consent"))
	fmt.Printf("Открой в браузере и введи код:\n%v\nКод: ", authURL)
	var code string
	if _, err := fmt.Scanln(&code); err != nil {
		return nil, fmt.Errorf("ввод кода: %w", err)
	}
	return config.Exchange(ctx, strings.TrimSpace(code))
}

func saveToken(path string, token *oauth2.Token) error {
	f, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		return err
	}
	defer f.Close()
	return json.NewEncoder(f).Encode(token)
}

// ListRecursive возвращает метаданные всех файлов в папке и подпапках (ярлыки пропускаются).
func (d *DriveSource) ListRecursive(ctx context.Context) ([]FileMeta, error) {
	var out []FileMeta
	type item struct {
		id   string
		path string
	}
	queue := []item{{id: d.folderID, path: ""}}
	pageSize := int64(100)
	fields := "nextPageToken, files(id, name, mimeType, size, modifiedTime, parents)"

	for len(queue) > 0 {
		cur := queue[0]
		queue = queue[1:]

		var pageToken string
		for {
			req := d.srv.Files.List().
				Q(fmt.Sprintf("'%s' in parents and trashed = false", cur.id)).
				PageSize(pageSize).
				Fields(googleapi.Field(fields))
			if pageToken != "" {
				req = req.PageToken(pageToken)
			}
			r, err := req.Do()
			if err != nil {
				return nil, fmt.Errorf("list %q: %w", cur.path, err)
			}
			for _, f := range r.Files {
				if f.MimeType == mimeTypeShortcut {
					continue
				}
				name := f.Name
				if name == "" {
					name = f.Id
				}
				relPath := name
				if cur.path != "" {
					relPath = cur.path + "/" + name
				}
				if f.MimeType == mimeTypeFolder {
					queue = append(queue, item{id: f.Id, path: relPath})
					continue
				}
				modifiedTime := ""
				if f.ModifiedTime != "" {
					modifiedTime = f.ModifiedTime
				}
				out = append(out, FileMeta{
					ID:           f.Id,
					Name:         f.Name,
					MimeType:     f.MimeType,
					Size:         f.Size,
					ModifiedTime: modifiedTime,
					Path:         relPath,
					Parents:      f.Parents,
				})
			}
			pageToken = r.NextPageToken
			if pageToken == "" {
				break
			}
		}
	}
	return out, nil
}

// DownloadToCache скачивает файл в .cache/downloads/<fileID>/; не качает повторно, если modifiedTime совпадает.
// Возвращает локальный путь к файлу и ошибку.
func (d *DriveSource) DownloadToCache(ctx context.Context, meta FileMeta) (localPath string, err error) {
	dir := filepath.Join(d.cacheDir, meta.ID)
	mtimePath := filepath.Join(dir, ".mtime")
	safeName := sanitizeName(meta.Name)
	useExport := strings.HasPrefix(meta.MimeType, "application/vnd.google-apps.")
	if useExport {
		safeName = strings.TrimSuffix(safeName, filepath.Ext(safeName)) + ".pdf"
	}
	filePath := filepath.Join(dir, safeName)

	if meta.ModifiedTime != "" {
		if b, _ := os.ReadFile(mtimePath); string(b) == meta.ModifiedTime {
			if _, err := os.Stat(filePath); err == nil {
				return filePath, nil
			}
		}
	}
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", fmt.Errorf("mkdir: %w", err)
	}
	var resp *http.Response
	if useExport {
		exportMime := exportMimeType(meta.MimeType)
		if exportMime == "" {
			return "", fmt.Errorf("unsupported Google type: %s", meta.MimeType)
		}
		resp, err = d.srv.Files.Export(meta.ID, exportMime).Context(ctx).Download()
	} else {
		resp, err = d.srv.Files.Get(meta.ID).Context(ctx).Download()
	}
	if err != nil {
		return "", fmt.Errorf("download %q: %w", meta.Name, err)
	}
	defer resp.Body.Close()
	f, err := os.Create(filePath)
	if err != nil {
		return "", fmt.Errorf("create file: %w", err)
	}
	_, err = io.Copy(f, resp.Body)
	f.Close()
	if err != nil {
		os.Remove(filePath)
		return "", fmt.Errorf("write: %w", err)
	}
	if meta.ModifiedTime != "" {
		_ = os.WriteFile(mtimePath, []byte(meta.ModifiedTime), 0644)
	}
	return filePath, nil
}

func sanitizeName(name string) string {
	return strings.Map(func(r rune) rune {
		if r == '/' || r == '\\' || r == 0 {
			return '_'
		}
		return r
	}, name)
}

// exportMimeType возвращает MIME для Export Google Docs/Sheets и т.д.
func exportMimeType(googleMime string) string {
	switch googleMime {
	case "application/vnd.google-apps.document":
		return "application/pdf"
	case "application/vnd.google-apps.spreadsheet":
		return "application/pdf"
	case "application/vnd.google-apps.presentation":
		return "application/pdf"
	default:
		return ""
	}
}
