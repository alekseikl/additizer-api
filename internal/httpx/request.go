package httpx

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
)

func DecodeJSON[T any](r *http.Request) (T, error) {
	var v T
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(&v); err != nil {
		return v, errors.New("invalid json body")
	}
	return v, nil
}

func UintURLParam(w http.ResponseWriter, r *http.Request, name string) (uint, bool) {
	value, err := strconv.ParseUint(chi.URLParam(r, name), 10, 0)
	if err != nil || value == 0 {
		WriteError(w, http.StatusBadRequest, "invalid "+name)
		return 0, false
	}
	return uint(value), true
}
