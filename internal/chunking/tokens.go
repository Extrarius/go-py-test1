package chunking

import "unicode/utf8"

// CharsPerToken — эвристика: ~4 символа на 1 токен (как у многих моделей).
const CharsPerToken = 4

// EstimateTokens возвращает примерное число токенов (руны / 4).
func EstimateTokens(s string) int {
	n := utf8.RuneCountInString(s)
	if n <= 0 {
		return 0
	}
	return (n + CharsPerToken - 1) / CharsPerToken
}
