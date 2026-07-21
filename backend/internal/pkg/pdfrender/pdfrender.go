package pdfrender

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

const DefaultDPI = 160

// PagePNG is a rendered PDF page.
type PagePNG struct {
	Number int
	PNG    []byte
}

// Available reports whether pdftoppm is on PATH.
func Available() bool {
	_, err := exec.LookPath("pdftoppm")
	return err == nil
}

// RenderAll writes the PDF to a temp file and renders each page to PNG via pdftoppm.
func RenderAll(pdf []byte, dpi int) ([]PagePNG, error) {
	if len(pdf) == 0 {
		return nil, fmt.Errorf("pdfrender: empty pdf")
	}
	if dpi < 72 {
		dpi = DefaultDPI
	}
	if dpi > 300 {
		dpi = 300
	}
	if !Available() {
		return nil, fmt.Errorf("pdfrender: pdftoppm not found (install poppler-utils)")
	}

	dir, err := os.MkdirTemp("", "asutport-pdf-*")
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(dir)

	src := filepath.Join(dir, "input.pdf")
	if err := os.WriteFile(src, pdf, 0o600); err != nil {
		return nil, err
	}
	prefix := filepath.Join(dir, "page")
	cmd := exec.Command("pdftoppm", "-png", "-r", strconv.Itoa(dpi), src, prefix)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("pdftoppm: %w (%s)", err, strings.TrimSpace(string(out)))
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	var names []string
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		if strings.HasPrefix(name, "page") && strings.HasSuffix(name, ".png") {
			names = append(names, name)
		}
	}
	sort.Strings(names)
	if len(names) == 0 {
		return nil, fmt.Errorf("pdfrender: no pages rendered")
	}

	pages := make([]PagePNG, 0, len(names))
	for i, name := range names {
		data, err := os.ReadFile(filepath.Join(dir, name))
		if err != nil {
			return nil, err
		}
		pages = append(pages, PagePNG{Number: i + 1, PNG: data})
	}
	return pages, nil
}
