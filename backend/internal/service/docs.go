package service

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"log/slog"
	"path/filepath"
	"regexp"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/google/uuid"

	"github.com/n8node/asutport/internal/llm"
	"github.com/n8node/asutport/internal/pkg/chunker"
	"github.com/n8node/asutport/internal/pkg/pdfextract"
	"github.com/n8node/asutport/internal/pkg/pdfrender"
	"github.com/n8node/asutport/internal/repository"
	s3store "github.com/n8node/asutport/internal/s3"
)

const (
	embedBatchSize     = 32
	maxUploadBytes     = 100 << 20 // 100 MiB
	minLocalPageRunes  = 80
	maxVisionPagesHint = 500 // soft guard for runaway cost
)

var slugRe = regexp.MustCompile(`[^a-z0-9-]+`)

type DocsService struct {
	docs   *repository.DocsRepo
	orgs   *repository.OrgRepo
	s3     *s3store.Loader
	llm    *llm.Resolver
	logger *slog.Logger
	queue  chan uuid.UUID
}

func NewDocsService(
	docs *repository.DocsRepo,
	orgs *repository.OrgRepo,
	s3 *s3store.Loader,
	llmResolver *llm.Resolver,
	logger *slog.Logger,
) *DocsService {
	s := &DocsService{
		docs:   docs,
		orgs:   orgs,
		s3:     s3,
		llm:    llmResolver,
		logger: logger,
		queue:  make(chan uuid.UUID, 64),
	}
	go s.workerLoop()
	return s
}

func (s *DocsService) Enqueue(sourceID uuid.UUID) {
	select {
	case s.queue <- sourceID:
	default:
		go func() { s.queue <- sourceID }()
	}
}

func (s *DocsService) workerLoop() {
	for id := range s.queue {
		ctx := context.Background()
		if err := s.ProcessSource(ctx, id); err != nil {
			s.logger.Error("doc process", slog.String("source_id", id.String()), slog.Any("err", err))
		}
	}
}

type UploadInput struct {
	ManufacturerOrgID uuid.UUID
	ProductID         uuid.UUID
	Version           string
	Filename          string
	MimeType          string
	Data              []byte
}

func (s *DocsService) Upload(ctx context.Context, in UploadInput) (*repository.DocSource, error) {
	if len(in.Data) == 0 {
		return nil, fmt.Errorf("empty file")
	}
	if len(in.Data) > maxUploadBytes {
		return nil, fmt.Errorf("file exceeds 100 MB limit")
	}
	product, err := s.docs.GetProduct(ctx, in.ProductID)
	if err != nil {
		return nil, err
	}
	if product.ManufacturerOrgID != in.ManufacturerOrgID {
		return nil, fmt.Errorf("product does not belong to organization")
	}
	org, err := s.orgs.GetByID(ctx, in.ManufacturerOrgID)
	if err != nil {
		return nil, err
	}
	hash := contentHashBytes(in.Data)
	if existing, err := s.docs.FindByContentHash(ctx, in.ManufacturerOrgID, hash); err == nil && existing != nil {
		dup := &repository.DocSource{
			ProductID:         in.ProductID,
			ManufacturerOrgID: in.ManufacturerOrgID,
			Version:           strings.TrimSpace(in.Version),
			Filename:          sanitizeDocFilename(in.Filename),
			MimeType:          firstMime(in.MimeType, in.Filename),
			ByteSize:          int64(len(in.Data)),
			S3OriginalKey:     existing.S3OriginalKey,
			ContentHash:       hash,
			Status:            "skipped_duplicate",
			PageCount:         existing.PageCount,
			ChunkCount:        existing.ChunkCount,
			EmbeddingModel:    existing.EmbeddingModel,
			TokensTotal:       0,
			ErrorMessage:      "identical content already indexed: " + existing.ID.String(),
		}
		if err := s.docs.CreateDocSource(ctx, dup); err != nil {
			return nil, err
		}
		return dup, nil
	}

	mfrSlug := strings.TrimSpace(org.Slug)
	if mfrSlug == "" {
		mfrSlug = slugify(org.Name)
	}
	if mfrSlug == "" {
		mfrSlug = in.ManufacturerOrgID.String()[:8]
	}
	version := strings.TrimSpace(in.Version)
	if version == "" {
		version = "unspecified"
	}
	filename := sanitizeDocFilename(in.Filename)
	key := s3store.DocOriginalKey(mfrSlug, product.Slug, version, filename)

	client, err := s.s3.Client(ctx)
	if err != nil {
		return nil, fmt.Errorf("object storage: %w", err)
	}
	mime := firstMime(in.MimeType, filename)
	if err := client.PutObject(ctx, key, mime, bytes.NewReader(in.Data), int64(len(in.Data))); err != nil {
		return nil, fmt.Errorf("upload to storage: %w", err)
	}

	src := &repository.DocSource{
		ProductID:         in.ProductID,
		ManufacturerOrgID: in.ManufacturerOrgID,
		Version:           version,
		Filename:          filename,
		MimeType:          mime,
		ByteSize:          int64(len(in.Data)),
		S3OriginalKey:     key,
		ContentHash:       hash,
		Status:            "pending",
	}
	if err := s.docs.CreateDocSource(ctx, src); err != nil {
		return nil, err
	}
	s.Enqueue(src.ID)
	return src, nil
}

type pendingChunk struct {
	page      int
	index     int
	text      string
	tokens    int
	s3PageKey string
}

func (s *DocsService) ProcessSource(ctx context.Context, sourceID uuid.UUID) error {
	src, err := s.docs.GetDocSource(ctx, sourceID)
	if err != nil {
		return err
	}
	if src.Status == "ready" || src.Status == "skipped_duplicate" {
		return nil
	}
	_ = s.docs.UpdateDocSourceStatus(ctx, sourceID, "extracting", "")

	s3c, err := s.s3.Client(ctx)
	if err != nil {
		_ = s.docs.UpdateDocSourceStatus(ctx, sourceID, "failed", err.Error())
		return err
	}
	data, err := s3c.GetObjectBytes(ctx, src.S3OriginalKey)
	if err != nil {
		_ = s.docs.UpdateDocSourceStatus(ctx, sourceID, "failed", err.Error())
		return err
	}

	mfrSlug, productSlug, version := parseDocKeyParts(src.S3OriginalKey, src.ProductSlug, src.Version)

	pages, err := extractPages(src.Filename, src.MimeType, data)
	if err != nil {
		_ = s.docs.UpdateDocSourceStatus(ctx, sourceID, "failed", err.Error())
		return err
	}

	llmClient, resolved, err := s.llm.Client(ctx)
	if err != nil {
		_ = s.docs.UpdateDocSourceStatus(ctx, sourceID, "failed", err.Error())
		return err
	}

	isPDF := strings.HasSuffix(strings.ToLower(src.Filename), ".pdf") || strings.Contains(src.MimeType, "pdf")
	var rendered map[int][]byte
	if isPDF && pdfrender.Available() {
		pngs, rerr := pdfrender.RenderAll(data, pdfrender.DefaultDPI)
		if rerr != nil {
			s.logger.Warn("pdf render", slog.String("source_id", sourceID.String()), slog.Any("err", rerr))
		} else {
			rendered = make(map[int][]byte, len(pngs))
			for _, p := range pngs {
				rendered[p.Number] = p.PNG
			}
			// Ensure page list covers rendered pages even if text extract missed blanks.
			if len(pngs) > len(pages) {
				seen := map[int]bool{}
				for _, p := range pages {
					seen[p.Number] = true
				}
				for _, p := range pngs {
					if !seen[p.Number] {
						pages = append(pages, pageText{Number: p.Number, Text: ""})
					}
				}
			}
		}
	}

	_ = s.docs.DeletePagesForSource(ctx, sourceID)
	_ = s.docs.DeleteChunksForSource(ctx, sourceID)

	var pending []pendingChunk
	var visionTokens int64
	visionPages := 0

	for _, page := range pages {
		text := strings.TrimSpace(page.Text)
		textSource := "local"
		pageKey := ""
		parsedKey := ""

		if png, ok := rendered[page.Number]; ok && len(png) > 0 {
			pageKey = s3store.DocPageKey(mfrSlug, productSlug, version, page.Number)
			if err := s3c.PutObject(ctx, pageKey, "image/png", bytes.NewReader(png), int64(len(png))); err != nil {
				_ = s.docs.UpdateDocSourceStatus(ctx, sourceID, "failed", err.Error())
				return err
			}
			needVision := utf8.RuneCountInString(text) < minLocalPageRunes
			if needVision {
				if visionPages >= maxVisionPagesHint {
					s.logger.Warn("vision page limit", slog.String("source_id", sourceID.String()), slog.Int("page", page.Number))
				} else {
					md, usage, verr := llmClient.ParsePageImage(ctx, resolved.VisionModel, page.Number, png)
					if verr != nil {
						s.logger.Warn("vision page", slog.Int("page", page.Number), slog.Any("err", verr))
					} else if strings.TrimSpace(md) != "" {
						text = strings.TrimSpace(md)
						textSource = "vision"
						visionTokens += usage.TotalTokens
						visionPages++
						parsedKey = s3store.DocParsedKey(mfrSlug, productSlug, version, page.Number)
						_ = s3c.PutObject(ctx, parsedKey, "text/markdown; charset=utf-8", strings.NewReader(text), int64(len(text)))
						orgID := src.ManufacturerOrgID
						_ = s.docs.LogUsage(ctx, &orgID, "vision_parse", resolved.VisionModel, usage.PromptTokens, usage.TotalTokens, &sourceID, map[string]any{
							"page": page.Number,
						})
					}
				}
			}
		}

		_ = s.docs.UpsertPage(ctx, repository.DocPage{
			DocSourceID: sourceID,
			PageNumber:  page.Number,
			S3PageKey:   pageKey,
			S3ParsedKey: parsedKey,
			TextSource:  textSource,
			CharCount:   utf8.RuneCountInString(text),
		})

		if text == "" {
			continue
		}
		parts := chunker.Split(text, chunker.DefaultSize, chunker.DefaultOverlap)
		for _, part := range parts {
			pending = append(pending, pendingChunk{
				page:      page.Number,
				index:     part.Index,
				text:      part.Text,
				tokens:    chunker.EstimateTokens(part.Text),
				s3PageKey: pageKey,
			})
		}
	}

	if len(pending) == 0 {
		msg := "no text chunks"
		if isPDF && !pdfrender.Available() {
			msg = "no text chunks; install poppler-utils for page render + vision fallback"
		} else if isPDF {
			msg = "no text chunks after local extract and vision"
		}
		_ = s.docs.UpdateDocSourceStatus(ctx, sourceID, "failed", msg)
		return fmt.Errorf("%s", msg)
	}

	_ = s.docs.UpdateDocSourceStatus(ctx, sourceID, "embedding", "")
	embedModel := resolved.EmbedModel
	var embedTokens int64
	for i := 0; i < len(pending); i += embedBatchSize {
		j := i + embedBatchSize
		if j > len(pending) {
			j = len(pending)
		}
		batch := pending[i:j]
		texts := make([]string, len(batch))
		for k, c := range batch {
			texts[k] = c.text
		}
		vecs, tokens, err := llmClient.Embed(ctx, embedModel, texts)
		if err != nil {
			_ = s.docs.UpdateDocSourceStatus(ctx, sourceID, "failed", err.Error())
			return err
		}
		embedTokens += int64(tokens)
		for k, c := range batch {
			_, err := s.docs.InsertChunk(ctx, repository.DocChunk{
				DocSourceID:       sourceID,
				ProductID:         src.ProductID,
				ManufacturerOrgID: src.ManufacturerOrgID,
				Version:           src.Version,
				PageNumber:        c.page,
				ChunkIndex:        c.index,
				ContentMD:         c.text,
				S3PageKey:         c.s3PageKey,
				TokenEstimate:     c.tokens,
			}, vectorLiteral(vecs[k]))
			if err != nil {
				_ = s.docs.UpdateDocSourceStatus(ctx, sourceID, "failed", err.Error())
				return err
			}
		}
	}

	totalTokens := embedTokens + visionTokens
	orgID := src.ManufacturerOrgID
	_ = s.docs.LogUsage(ctx, &orgID, "embedding", embedModel, embedTokens, embedTokens, &sourceID, map[string]any{
		"chunks":       len(pending),
		"pages":        len(pages),
		"vision_pages": visionPages,
	})

	if err := s.docs.MarkDocSourceReady(ctx, sourceID, len(pages), len(pending), embedModel, totalTokens); err != nil {
		return err
	}
	s.logger.Info("doc indexed",
		slog.String("source_id", sourceID.String()),
		slog.Int("chunks", len(pending)),
		slog.Int("vision_pages", visionPages),
		slog.Int64("tokens", totalTokens),
	)
	return nil
}

func (s *DocsService) PagePresignURL(ctx context.Context, sourceID uuid.UUID, page int, orgID uuid.UUID, superadmin bool) (string, error) {
	src, err := s.docs.GetDocSource(ctx, sourceID)
	if err != nil {
		return "", err
	}
	if !superadmin && src.ManufacturerOrgID != orgID {
		return "", fmt.Errorf("access denied")
	}
	pg, err := s.docs.GetPage(ctx, sourceID, page)
	if err != nil {
		return "", err
	}
	if strings.TrimSpace(pg.S3PageKey) == "" {
		return "", fmt.Errorf("page image not available")
	}
	client, err := s.s3.Client(ctx)
	if err != nil {
		return "", err
	}
	return client.PresignGet(ctx, pg.S3PageKey, 0)
}

type SearchInput struct {
	Query      string
	ProductIDs []uuid.UUID
	Version    string
	TopK       int
	Threshold  float64
	OrgID      *uuid.UUID
}

func (s *DocsService) Search(ctx context.Context, in SearchInput) ([]repository.RAGHit, error) {
	q := strings.TrimSpace(in.Query)
	if len(q) < 2 {
		return nil, fmt.Errorf("query too short")
	}
	topK := in.TopK
	if topK <= 0 {
		topK = 10
	}
	if topK > 50 {
		topK = 50
	}
	threshold := in.Threshold
	if threshold <= 0 {
		threshold = 0.45
	}

	kw, err := s.docs.SearchKeyword(ctx, q, in.ProductIDs, in.Version, topK)
	if err != nil {
		return nil, err
	}

	llmClient, resolved, err := s.llm.Client(ctx)
	var sem []repository.RAGHit
	if err == nil {
		vec, tokens, eerr := llmClient.EmbedOne(ctx, resolved.EmbedModel, q)
		if eerr == nil {
			_ = s.docs.LogUsage(ctx, in.OrgID, "rag_search", resolved.EmbedModel, int64(tokens), int64(tokens), nil, map[string]any{
				"query_len": len(q),
			})
			sem, err = s.docs.SearchSemantic(ctx, vectorLiteral(vec), in.ProductIDs, in.Version, topK, threshold)
			if err != nil {
				return nil, err
			}
		}
	}

	return mergeHits(kw, sem, topK), nil
}

func mergeHits(keyword, semantic []repository.RAGHit, limit int) []repository.RAGHit {
	seen := map[uuid.UUID]struct{}{}
	out := make([]repository.RAGHit, 0, limit)
	for _, h := range keyword {
		if _, ok := seen[h.ChunkID]; ok {
			continue
		}
		seen[h.ChunkID] = struct{}{}
		out = append(out, h)
		if len(out) >= limit {
			return out
		}
	}
	for _, h := range semantic {
		if _, ok := seen[h.ChunkID]; ok {
			continue
		}
		seen[h.ChunkID] = struct{}{}
		out = append(out, h)
		if len(out) >= limit {
			return out
		}
	}
	return out
}

type pageText struct {
	Number int
	Text   string
}

func extractPages(filename, mime string, data []byte) ([]pageText, error) {
	lower := strings.ToLower(filename)
	switch {
	case strings.HasSuffix(lower, ".md") || strings.HasSuffix(lower, ".txt") || strings.Contains(mime, "text/"):
		text := strings.TrimSpace(string(data))
		if text == "" {
			return nil, fmt.Errorf("empty text")
		}
		return []pageText{{Number: 1, Text: text}}, nil
	case strings.HasSuffix(lower, ".pdf") || strings.Contains(mime, "pdf"):
		pages, err := pdfextract.Pages(data)
		if err != nil {
			return nil, err
		}
		out := make([]pageText, 0, len(pages))
		for _, p := range pages {
			out = append(out, pageText{Number: p.Number, Text: p.Text})
		}
		return out, nil
	default:
		return nil, fmt.Errorf("unsupported file type (use PDF, MD, or TXT)")
	}
}

func parseDocKeyParts(s3Key, productSlug, version string) (mfr, product, ver string) {
	// docs/{mfr}/{product}/{version}/original/{file}
	parts := strings.Split(s3Key, "/")
	if len(parts) >= 4 && parts[0] == "docs" {
		return parts[1], parts[2], parts[3]
	}
	product = productSlug
	ver = version
	if ver == "" {
		ver = "unspecified"
	}
	if product == "" {
		product = "product"
	}
	return "org", product, ver
}

func contentHashBytes(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

func vectorLiteral(v []float32) string {
	var b strings.Builder
	b.Grow(len(v) * 12)
	b.WriteByte('[')
	for i, x := range v {
		if i > 0 {
			b.WriteByte(',')
		}
		fmt.Fprintf(&b, "%.9g", x)
	}
	b.WriteByte(']')
	return b.String()
}

func sanitizeDocFilename(name string) string {
	base := filepath.Base(strings.TrimSpace(name))
	base = strings.ReplaceAll(base, "..", "")
	var b strings.Builder
	for _, r := range base {
		if r < 32 || r == 127 {
			continue
		}
		b.WriteRune(r)
	}
	out := strings.TrimSpace(b.String())
	if out == "" {
		return "document.pdf"
	}
	return out
}

func slugify(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	var b strings.Builder
	for _, r := range s {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			b.WriteRune(r)
		} else if r == ' ' || r == '_' || r == '-' {
			b.WriteByte('-')
		}
	}
	out := slugRe.ReplaceAllString(b.String(), "-")
	out = strings.Trim(out, "-")
	if len(out) > 64 {
		out = out[:64]
	}
	return out
}

func firstMime(mime, filename string) string {
	if strings.TrimSpace(mime) != "" {
		return mime
	}
	lower := strings.ToLower(filename)
	switch {
	case strings.HasSuffix(lower, ".pdf"):
		return "application/pdf"
	case strings.HasSuffix(lower, ".md"):
		return "text/markdown"
	case strings.HasSuffix(lower, ".txt"):
		return "text/plain"
	default:
		return "application/octet-stream"
	}
}
