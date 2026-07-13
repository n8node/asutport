package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Health struct {
	Version string
	Pool    *pgxpool.Pool
	S3      interface {
		Ping(context.Context) error
	}
}

func NewHealth(version string, pool *pgxpool.Pool, s3Client interface{ Ping(context.Context) error }) *Health {
	return &Health{Version: version, Pool: pool, S3: s3Client}
}

func (h *Health) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
		return
	}

	pg := "ok"
	if h.Pool != nil {
		c, cancel := context.WithTimeout(r.Context(), 2*time.Second)
		defer cancel()
		if err := h.Pool.Ping(c); err != nil {
			pg = "error"
		}
	}

	s3Status := "ok"
	if h.S3 != nil {
		c, cancel := context.WithTimeout(r.Context(), 3*time.Second)
		defer cancel()
		if err := h.S3.Ping(c); err != nil {
			s3Status = "error"
		}
	} else {
		s3Status = "error"
	}

	status := "ok"
	if pg != "ok" || s3Status != "ok" {
		status = "degraded"
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]string{
		"status":   status,
		"version":  h.Version,
		"postgres": pg,
		"s3":       s3Status,
	})
}
