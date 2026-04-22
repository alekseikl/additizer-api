package handlers

import (
	"net/http"

	"github.com/alekseikl/additizer-api/internal/httpx"
)

func Health(w http.ResponseWriter, r *http.Request) {
	httpx.WriteJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}
