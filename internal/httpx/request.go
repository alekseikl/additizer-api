package httpx

import (
	"encoding/json"
	"errors"
	"net/http"
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
