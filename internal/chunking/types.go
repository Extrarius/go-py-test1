package chunking

// Chunk — фрагмент текста с метаданными для саммаризации.
type Chunk struct {
	Text       string // текст чанка
	FileID     string // источник (file_id)
	ChunkIndex int    // порядковый номер чанка в документе (0-based)
	StartRune  int    // начальная позиция в исходном документе (в рунах)
	EndRune    int    // конечная позиция в исходном документе
	TokenEst   int    // примерная оценка числа токенов
}
