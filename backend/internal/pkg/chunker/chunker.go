package chunker

import (
	"strings"
	"unicode/utf8"
)

const (
	DefaultSize    = 1200
	DefaultOverlap = 150
)

type Chunk struct {
	Text      string
	Index     int
	StartChar int
	EndChar   int
}

// Split splits text on paragraph-aware windows (characters, not tokens).
func Split(text string, size, overlap int) []Chunk {
	cleaned := strings.TrimSpace(strings.ReplaceAll(text, "\r\n", "\n"))
	for strings.Contains(cleaned, "\n\n\n") {
		cleaned = strings.ReplaceAll(cleaned, "\n\n\n", "\n\n")
	}
	if cleaned == "" {
		return nil
	}
	if size < 200 {
		size = DefaultSize
	}
	if overlap < 0 {
		overlap = 0
	}
	if overlap >= size {
		overlap = size / 5
	}
	if utf8.RuneCountInString(cleaned) <= size {
		return []Chunk{{Text: cleaned, Index: 0, StartChar: 0, EndChar: len(cleaned)}}
	}

	var out []Chunk
	start := 0
	idx := 0
	runes := []rune(cleaned)
	for start < len(runes) {
		end := start + size
		if end > len(runes) {
			end = len(runes)
		}
		if end < len(runes) {
			end = breakPoint(runes, start, end)
		}
		chunk := strings.TrimSpace(string(runes[start:end]))
		if chunk != "" {
			out = append(out, Chunk{
				Text:      chunk,
				Index:     idx,
				StartChar: start,
				EndChar:   end,
			})
			idx++
		}
		if end >= len(runes) {
			break
		}
		next := end - overlap
		if next <= start {
			next = start + 1
		}
		start = next
	}
	return out
}

func breakPoint(runes []rune, start, end int) int {
	windowStart := end - 80
	if windowStart < start {
		windowStart = start
	}
	window := string(runes[windowStart:end])
	if i := strings.LastIndex(window, "\n\n"); i > 0 {
		return windowStart + i + 2
	}
	for _, sep := range []string{". ", "! ", "? ", ".\n", "!\n", "?\n"} {
		if i := strings.LastIndex(window, sep); i > 0 {
			return windowStart + i + len(sep)
		}
	}
	if i := strings.LastIndex(window, " "); i > 0 {
		return windowStart + i + 1
	}
	return end
}

func EstimateTokens(s string) int {
	n := utf8.RuneCountInString(s)
	if n == 0 {
		return 1
	}
	t := n / 4
	if t < 1 {
		t = 1
	}
	return t
}
