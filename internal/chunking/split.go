package chunking

import (
	"strings"
	"unicode/utf8"
)

// Options настройки разбиения на чанки.
type Options struct {
	MaxTokens      int // макс. токенов на чанк (например 2000–4000)
	OverlapPercent int // перекрытие между чанками 5–10
}

// DefaultOptions возвращает разумные значения по умолчанию.
func DefaultOptions() Options {
	return Options{MaxTokens: 4000, OverlapPercent: 10}
}

// Split разбивает текст документа на чанки. fileID — идентификатор источника (например Drive file_id).
func Split(text string, fileID string, opts Options) []Chunk {
	if opts.MaxTokens <= 0 {
		opts.MaxTokens = 4000
	}
	if opts.OverlapPercent < 0 {
		opts.OverlapPercent = 0
	}
	if opts.OverlapPercent > 50 {
		opts.OverlapPercent = 50
	}
	segments := splitIntoSegments(text, opts.MaxTokens*CharsPerToken)
	return buildChunks(segments, fileID, opts)
}

type segment struct {
	startRune, endRune int
	text              string
}

// splitIntoSegments режет текст по абзацам (\n\n), при необходимости по строкам; maxRunes — макс. размер сегмента в рунах.
func splitIntoSegments(text string, maxRunes int) []segment {
	var out []segment
	if maxRunes <= 0 {
		maxRunes = 4000 * CharsPerToken
	}
	paragraphs := strings.Split(text, "\n\n")
	pos := 0
	for _, p := range paragraphs {
		trimmed := strings.TrimSpace(p)
		if trimmed == "" {
			pos += utf8.RuneCountInString(p) + 2
			continue
		}
		pStart := pos
		pRunes := utf8.RuneCountInString(trimmed)
		pEnd := pStart + pRunes
		if pRunes <= maxRunes {
			out = append(out, segment{startRune: pStart, endRune: pEnd, text: trimmed})
		} else {
			lines := strings.Split(trimmed, "\n")
			lineStart := pStart
			for _, line := range lines {
				lineRunes := utf8.RuneCountInString(line)
				lineEnd := lineStart + lineRunes
				out = append(out, segment{startRune: lineStart, endRune: lineEnd, text: line})
				lineStart = lineEnd + 1
			}
		}
		pos = pEnd + 2
	}
	return out
}

func buildChunks(segments []segment, fileID string, opts Options) []Chunk {
	var chunks []Chunk
	overlapTokens := opts.MaxTokens * opts.OverlapPercent / 100
	if overlapTokens >= opts.MaxTokens {
		overlapTokens = 0
	}
	overlapRunes := overlapTokens * CharsPerToken

	var curParts []string
	var curStart, curEnd int
	curTokens := 0
	chunkIndex := 0

	for _, seg := range segments {
		segTokens := EstimateTokens(seg.text)
		if curTokens+segTokens > opts.MaxTokens && len(curParts) > 0 {
			flush := strings.Join(curParts, "\n\n")
			chunks = append(chunks, Chunk{
				Text:       flush,
				FileID:     fileID,
				ChunkIndex: chunkIndex,
				StartRune:  curStart,
				EndRune:    curEnd,
				TokenEst:   EstimateTokens(flush),
			})
			chunkIndex++
			curParts = nil
			curTokens = 0
			if overlapRunes > 0 && len(flush) > 0 {
				overlapStr := runeSuffix(flush, overlapRunes)
				curParts = []string{overlapStr}
				curTokens = EstimateTokens(overlapStr)
				curStart = curEnd - utf8.RuneCountInString(overlapStr)
				curEnd = curStart + utf8.RuneCountInString(overlapStr)
			} else {
				curStart = seg.startRune
				curEnd = seg.endRune
			}
		} else if len(curParts) == 0 {
			curStart = seg.startRune
			curEnd = seg.endRune
		} else {
			curEnd = seg.endRune
		}
		curParts = append(curParts, seg.text)
		curTokens = EstimateTokens(strings.Join(curParts, "\n\n"))
	}

	if len(curParts) > 0 {
		flush := strings.Join(curParts, "\n\n")
		chunks = append(chunks, Chunk{
			Text:       flush,
			FileID:     fileID,
			ChunkIndex: chunkIndex,
			StartRune:  curStart,
			EndRune:    curEnd,
			TokenEst:   EstimateTokens(flush),
		})
	}
	return chunks
}

func runeSuffix(s string, n int) string {
	runes := []rune(s)
	if n >= len(runes) {
		return s
	}
	return string(runes[len(runes)-n:])
}
