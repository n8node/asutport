package pdfextract

import (
	"bytes"
	"fmt"
	"strings"

	"github.com/ledongthuc/pdf"
)

type Page struct {
	Number int
	Text   string
}

// Pages extracts plain text per PDF page. Scanned pages may return empty text.
func Pages(data []byte) ([]Page, error) {
	r, err := pdf.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return nil, fmt.Errorf("pdf open: %w", err)
	}
	n := r.NumPage()
	if n <= 0 {
		return nil, fmt.Errorf("pdf: no pages")
	}
	out := make([]Page, 0, n)
	for i := 1; i <= n; i++ {
		p := r.Page(i)
		if p.V.IsNull() {
			out = append(out, Page{Number: i, Text: ""})
			continue
		}
		text, err := p.GetPlainText(nil)
		if err != nil {
			out = append(out, Page{Number: i, Text: ""})
			continue
		}
		out = append(out, Page{Number: i, Text: collapse(text)})
	}
	return out, nil
}

func collapse(s string) string {
	s = strings.ReplaceAll(s, "\r\n", "\n")
	s = strings.TrimSpace(s)
	for strings.Contains(s, "  ") {
		s = strings.ReplaceAll(s, "  ", " ")
	}
	for strings.Contains(s, "\n\n\n") {
		s = strings.ReplaceAll(s, "\n\n\n", "\n\n")
	}
	return s
}
