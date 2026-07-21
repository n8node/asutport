package handler

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/n8node/asutport/internal/auth"
	"github.com/n8node/asutport/internal/repository"
	s3store "github.com/n8node/asutport/internal/s3"
	"github.com/n8node/asutport/internal/service"
)

type DocsHandler struct {
	docs *service.DocsService
	repo *repository.DocsRepo
	orgs *repository.OrgRepo
	s3   *s3store.Loader
}

func NewDocsHandler(docs *service.DocsService, repo *repository.DocsRepo, orgs *repository.OrgRepo, s3 *s3store.Loader) *DocsHandler {
	return &DocsHandler{docs: docs, repo: repo, orgs: orgs, s3: s3}
}

type createProductRequest struct {
	ManufacturerOrgID string `json:"manufacturer_org_id"`
	Slug              string `json:"slug"`
	Name              string `json:"name"`
	Kind              string `json:"kind"`
	Description       string `json:"description"`
}

func (h *DocsHandler) CreateProduct(w http.ResponseWriter, r *http.Request) {
	p, ok := auth.PrincipalFromContext(r.Context())
	if !ok {
		WriteError(w, http.StatusUnauthorized, "UNAUTHORIZED", "missing authentication")
		return
	}
	var req createProductRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteError(w, http.StatusBadRequest, "INVALID_JSON", "invalid json")
		return
	}
	orgID := p.OrgID
	if p.IsSuperAdmin() && strings.TrimSpace(req.ManufacturerOrgID) != "" {
		parsed, err := uuid.Parse(strings.TrimSpace(req.ManufacturerOrgID))
		if err != nil {
			WriteError(w, http.StatusBadRequest, "VALIDATION_ERROR", "invalid manufacturer_org_id")
			return
		}
		orgID = parsed
	}
	org, err := h.orgs.GetByID(r.Context(), orgID)
	if err != nil {
		WriteError(w, http.StatusNotFound, "NOT_FOUND", "organization not found")
		return
	}
	if !p.IsSuperAdmin() && org.Type != "manufacturer" {
		WriteError(w, http.StatusForbidden, "FORBIDDEN", "only manufacturer organizations can create products")
		return
	}
	kind := strings.TrimSpace(req.Kind)
	if kind == "" {
		kind = "other"
	}
	slug := strings.TrimSpace(req.Slug)
	name := strings.TrimSpace(req.Name)
	if slug == "" || name == "" {
		WriteError(w, http.StatusBadRequest, "VALIDATION_ERROR", "slug and name are required")
		return
	}
	product, err := h.repo.CreateProduct(r.Context(), orgID, slug, name, kind, strings.TrimSpace(req.Description))
	if err != nil {
		WriteError(w, http.StatusBadRequest, "VALIDATION_ERROR", "could not create product (check slug uniqueness and kind)")
		return
	}
	WriteJSON(w, http.StatusCreated, map[string]any{"data": product})
}

func (h *DocsHandler) ListProducts(w http.ResponseWriter, r *http.Request) {
	p, ok := auth.PrincipalFromContext(r.Context())
	if !ok {
		WriteError(w, http.StatusUnauthorized, "UNAUTHORIZED", "missing authentication")
		return
	}
	orgID := p.OrgID
	if p.IsSuperAdmin() {
		if q := strings.TrimSpace(r.URL.Query().Get("manufacturer_org_id")); q != "" {
			parsed, err := uuid.Parse(q)
			if err != nil {
				WriteError(w, http.StatusBadRequest, "VALIDATION_ERROR", "invalid manufacturer_org_id")
				return
			}
			orgID = parsed
		}
	}
	items, err := h.repo.ListProducts(r.Context(), orgID)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "INTERNAL", "failed to list products")
		return
	}
	WriteJSON(w, http.StatusOK, map[string]any{"data": items})
}

func (h *DocsHandler) ListSources(w http.ResponseWriter, r *http.Request) {
	p, ok := auth.PrincipalFromContext(r.Context())
	if !ok {
		WriteError(w, http.StatusUnauthorized, "UNAUTHORIZED", "missing authentication")
		return
	}
	var orgFilter *uuid.UUID
	if !p.IsSuperAdmin() {
		id := p.OrgID
		orgFilter = &id
	} else if q := strings.TrimSpace(r.URL.Query().Get("manufacturer_org_id")); q != "" {
		parsed, err := uuid.Parse(q)
		if err != nil {
			WriteError(w, http.StatusBadRequest, "VALIDATION_ERROR", "invalid manufacturer_org_id")
			return
		}
		orgFilter = &parsed
	}
	items, err := h.repo.ListDocSources(r.Context(), orgFilter, 100)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "INTERNAL", "failed to list documents")
		return
	}
	WriteJSON(w, http.StatusOK, map[string]any{"data": items})
}

func (h *DocsHandler) GetSource(w http.ResponseWriter, r *http.Request) {
	p, ok := auth.PrincipalFromContext(r.Context())
	if !ok {
		WriteError(w, http.StatusUnauthorized, "UNAUTHORIZED", "missing authentication")
		return
	}
	id, err := uuid.Parse(chi.URLParam(r, "sourceID"))
	if err != nil {
		WriteError(w, http.StatusBadRequest, "VALIDATION_ERROR", "invalid source id")
		return
	}
	src, err := h.repo.GetDocSource(r.Context(), id)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			WriteError(w, http.StatusNotFound, "NOT_FOUND", "document not found")
			return
		}
		WriteError(w, http.StatusInternalServerError, "INTERNAL", "failed to load document")
		return
	}
	if !p.IsSuperAdmin() && src.ManufacturerOrgID != p.OrgID {
		WriteError(w, http.StatusForbidden, "FORBIDDEN", "access denied")
		return
	}
	WriteJSON(w, http.StatusOK, map[string]any{"data": src})
}

func (h *DocsHandler) Upload(w http.ResponseWriter, r *http.Request) {
	p, ok := auth.PrincipalFromContext(r.Context())
	if !ok {
		WriteError(w, http.StatusUnauthorized, "UNAUTHORIZED", "missing authentication")
		return
	}
	const maxMem = 32 << 20
	if err := r.ParseMultipartForm(maxMem); err != nil {
		WriteError(w, http.StatusBadRequest, "VALIDATION_ERROR", "multipart form required")
		return
	}
	productID, err := uuid.Parse(strings.TrimSpace(r.FormValue("product_id")))
	if err != nil {
		WriteError(w, http.StatusBadRequest, "VALIDATION_ERROR", "product_id is required")
		return
	}
	version := strings.TrimSpace(r.FormValue("version"))
	file, header, err := r.FormFile("file")
	if err != nil {
		WriteError(w, http.StatusBadRequest, "VALIDATION_ERROR", "file is required")
		return
	}
	defer file.Close()
	data, err := io.ReadAll(io.LimitReader(file, 100<<20+1))
	if err != nil {
		WriteError(w, http.StatusBadRequest, "VALIDATION_ERROR", "failed to read file")
		return
	}
	if len(data) > 100<<20 {
		WriteError(w, http.StatusBadRequest, "VALIDATION_ERROR", "file exceeds 100 MB")
		return
	}

	orgID := p.OrgID
	if p.IsSuperAdmin() {
		if q := strings.TrimSpace(r.FormValue("manufacturer_org_id")); q != "" {
			parsed, err := uuid.Parse(q)
			if err != nil {
				WriteError(w, http.StatusBadRequest, "VALIDATION_ERROR", "invalid manufacturer_org_id")
				return
			}
			orgID = parsed
		} else {
			product, err := h.repo.GetProduct(r.Context(), productID)
			if err != nil {
				WriteError(w, http.StatusNotFound, "NOT_FOUND", "product not found")
				return
			}
			orgID = product.ManufacturerOrgID
		}
	}

	mime := header.Header.Get("Content-Type")
	src, err := h.docs.Upload(r.Context(), service.UploadInput{
		ManufacturerOrgID: orgID,
		ProductID:         productID,
		Version:           version,
		Filename:          header.Filename,
		MimeType:          mime,
		Data:              data,
	})
	if err != nil {
		WriteError(w, http.StatusBadRequest, "VALIDATION_ERROR", err.Error())
		return
	}
	WriteJSON(w, http.StatusAccepted, map[string]any{"data": src})
}

func (h *DocsHandler) Reprocess(w http.ResponseWriter, r *http.Request) {
	p, ok := auth.PrincipalFromContext(r.Context())
	if !ok {
		WriteError(w, http.StatusUnauthorized, "UNAUTHORIZED", "missing authentication")
		return
	}
	id, err := uuid.Parse(chi.URLParam(r, "sourceID"))
	if err != nil {
		WriteError(w, http.StatusBadRequest, "VALIDATION_ERROR", "invalid source id")
		return
	}
	src, err := h.repo.GetDocSource(r.Context(), id)
	if err != nil {
		WriteError(w, http.StatusNotFound, "NOT_FOUND", "document not found")
		return
	}
	if !p.IsSuperAdmin() && src.ManufacturerOrgID != p.OrgID {
		WriteError(w, http.StatusForbidden, "FORBIDDEN", "access denied")
		return
	}
	_ = h.repo.UpdateDocSourceStatus(r.Context(), id, "pending", "")
	h.docs.Enqueue(id)
	WriteJSON(w, http.StatusAccepted, map[string]any{"data": map[string]any{"id": id, "status": "pending"}})
}

type searchRequest struct {
	Query      string   `json:"query"`
	ProductIDs []string `json:"product_ids"`
	Version    string   `json:"version"`
	TopK       int      `json:"top_k"`
	Threshold  float64  `json:"threshold"`
}

func (h *DocsHandler) Search(w http.ResponseWriter, r *http.Request) {
	p, ok := auth.PrincipalFromContext(r.Context())
	if !ok {
		WriteError(w, http.StatusUnauthorized, "UNAUTHORIZED", "missing authentication")
		return
	}
	var req searchRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteError(w, http.StatusBadRequest, "INVALID_JSON", "invalid json")
		return
	}
	var productIDs []uuid.UUID
	for _, s := range req.ProductIDs {
		id, err := uuid.Parse(strings.TrimSpace(s))
		if err != nil {
			continue
		}
		productIDs = append(productIDs, id)
	}
	orgID := p.OrgID
	hits, err := h.docs.Search(r.Context(), service.SearchInput{
		Query:      req.Query,
		ProductIDs: productIDs,
		Version:    req.Version,
		TopK:       req.TopK,
		Threshold:  req.Threshold,
		OrgID:      &orgID,
	})
	if err != nil {
		WriteError(w, http.StatusBadRequest, "VALIDATION_ERROR", err.Error())
		return
	}
	WriteJSON(w, http.StatusOK, map[string]any{"data": hits})
}

func (h *DocsHandler) OriginalURL(w http.ResponseWriter, r *http.Request) {
	p, ok := auth.PrincipalFromContext(r.Context())
	if !ok {
		WriteError(w, http.StatusUnauthorized, "UNAUTHORIZED", "missing authentication")
		return
	}
	id, err := uuid.Parse(chi.URLParam(r, "sourceID"))
	if err != nil {
		WriteError(w, http.StatusBadRequest, "VALIDATION_ERROR", "invalid source id")
		return
	}
	src, err := h.repo.GetDocSource(r.Context(), id)
	if err != nil {
		WriteError(w, http.StatusNotFound, "NOT_FOUND", "document not found")
		return
	}
	if !p.IsSuperAdmin() && src.ManufacturerOrgID != p.OrgID {
		WriteError(w, http.StatusForbidden, "FORBIDDEN", "access denied")
		return
	}
	client, err := h.s3.Client(r.Context())
	if err != nil {
		WriteError(w, http.StatusServiceUnavailable, "STORAGE", "object storage unavailable")
		return
	}
	url, err := client.PresignGet(r.Context(), src.S3OriginalKey, 0)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "INTERNAL", "failed to presign")
		return
	}
	WriteJSON(w, http.StatusOK, map[string]any{
		"data": map[string]any{
			"url":         url,
			"expires_in":  3600,
			"source_id":   src.ID,
			"page_number": nil,
		},
	})
}
