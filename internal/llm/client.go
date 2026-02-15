package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"
)

const (
	openRouterURL      = "https://openrouter.ai/api/v1/chat/completions"
	defaultModel       = "deepseek/deepseek-v3.2" // на OpenRouter бывает $0; при 404 задай OPENROUTER_MODEL в .env
	defaultTimeout     = 120 * time.Second
	defaultConcurrency = 1 // для бесплатных моделей лучше 1, иначе 429
)

// Client — клиент OpenRouter с retry и лимитом параллелизма.
type Client struct {
	apiKey     string
	model      string
	httpClient *http.Client
	sem        chan struct{} // семафор для лимита параллельных запросов
}

// NewClient создаёт клиент. model пустой — подставляется defaultModel.
func NewClient(apiKey, model string, timeout time.Duration, maxConcurrency int) *Client {
	if model == "" {
		model = defaultModel
	}
	if timeout <= 0 {
		timeout = defaultTimeout
	}
	if maxConcurrency <= 0 {
		maxConcurrency = defaultConcurrency
	}
	return &Client{
		apiKey: apiKey,
		model:  model,
		httpClient: &http.Client{
			Timeout: timeout,
		},
		sem: make(chan struct{}, maxConcurrency),
	}
}

// Chat отправляет сообщения в модель и возвращает ответ (content первого choice).
func (c *Client) Chat(ctx context.Context, messages []Message) (string, error) {
	select {
	case c.sem <- struct{}{}:
		defer func() { <-c.sem }()
	case <-ctx.Done():
		return "", ctx.Err()
	}

	body := chatRequest{Model: c.model, Messages: messages}
	raw, err := json.Marshal(body)
	if err != nil {
		return "", fmt.Errorf("marshal: %w", err)
	}

	var lastErr error
	const maxAttempts = 10
	for attempt := 0; attempt < maxAttempts; attempt++ {
		if attempt > 0 {
			backoff := time.Duration(1<<uint(attempt-1)) * time.Second
			if backoff > 60*time.Second {
				backoff = 60 * time.Second
			}
			// при 429 (rate limit) ждём не меньше 30 с
			if attempt >= 4 && backoff < 30*time.Second {
				backoff = 30 * time.Second
			}
			log.Printf("llm retry %d after %v", attempt, backoff)
			select {
			case <-time.After(backoff):
			case <-ctx.Done():
				return "", ctx.Err()
			}
		}
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, openRouterURL, bytes.NewReader(raw))
		if err != nil {
			return "", fmt.Errorf("request: %w", err)
		}
		req.Header.Set("Authorization", "Bearer "+c.apiKey)
		req.Header.Set("Content-Type", "application/json")
		resp, err := c.httpClient.Do(req)
		if err != nil {
			lastErr = err
			continue
		}
		code := resp.StatusCode
		bodyBytes, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		if code == http.StatusOK {
			var out chatResponse
			if err := json.Unmarshal(bodyBytes, &out); err != nil {
				return "", fmt.Errorf("decode: %w", err)
			}
			if out.Error != nil {
				return "", fmt.Errorf("api error: %s", out.Error.Message)
			}
			if len(out.Choices) == 0 {
				return "", fmt.Errorf("no choices in response")
			}
			return out.Choices[0].Message.Content, nil
		}
		if code == 429 || code >= 500 {
			lastErr = fmt.Errorf("http %d: %s", code, string(bodyBytes))
			if code == 429 {
				log.Printf("rate limit (429), повтор через паузу; можно добавить свой ключ: https://openrouter.ai/settings/integrations")
			}
			continue
		}
		return "", fmt.Errorf("http %d: %s", code, string(bodyBytes))
	}
	return "", fmt.Errorf("retries exceeded: %w", lastErr)
}

// Model возвращает имя модели (для ключа кеша и т.п.).
func (c *Client) Model() string { return c.model }

// Acquire занимает слот параллелизма (для внешнего контроля). Release вызывать через defer.
func (c *Client) Acquire(ctx context.Context) (release func(), err error) {
	select {
	case c.sem <- struct{}{}:
		return func() { <-c.sem }, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}
