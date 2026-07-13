package handler

import (
	"encoding/json"
	"net/http"
)

type errBody struct {
	Error struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}

func WriteJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func WriteError(w http.ResponseWriter, status int, code, message string) {
	var b errBody
	b.Error.Code = code
	b.Error.Message = message
	WriteJSON(w, status, b)
}

func ClientIP(r *http.Request) string {
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		if i := len(xff); i > 0 {
			for idx := 0; idx < len(xff); idx++ {
				if xff[idx] == ',' {
					return xff[:idx]
				}
			}
			return xff
		}
	}
	if rip := r.Header.Get("X-Real-IP"); rip != "" {
		return rip
	}
	return r.RemoteAddr
}
