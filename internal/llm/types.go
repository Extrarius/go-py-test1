package llm

// Message — одно сообщение в диалоге (OpenAI-формат).
type Message struct {
	Role    string `json:"role"`    // "system", "user", "assistant"
	Content string `json:"content"`
}

// chatRequest — тело запроса к OpenRouter (OpenAI-совместимый).
type chatRequest struct {
	Model    string    `json:"model"`
	Messages []Message `json:"messages"`
}

// chatResponse — ответ chat/completions.
type chatResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error,omitempty"`
}
