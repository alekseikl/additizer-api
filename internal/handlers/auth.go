package handlers

import (
	"errors"
	"net/http"
	"time"

	"github.com/alekseikl/additizer-api/internal/httpx"
	"github.com/alekseikl/additizer-api/internal/middleware"
	"github.com/alekseikl/additizer-api/internal/users"
)

// "$(go env GOPATH)/bin/gorm" gen -i ./internal/models -o ./internal/generated
type AuthHandler struct {
	users *users.Service
}

func NewAuthHandler(users *users.Service) *AuthHandler {
	return &AuthHandler{users: users}
}

type registerRequest struct {
	Email     string `json:"email"`
	Username  string `json:"username"`
	FirstName string `json:"firstName"`
	LastName  string `json:"lastName"`
	Password  string `json:"password"`
}

type loginRequest struct {
	Identifier string `json:"identifier"`
	Password   string `json:"password"`
}

type userResponse struct {
	ID        string `json:"id"`
	Email     string `json:"email"`
	Username  string `json:"username"`
	FirstName string `json:"firstName"`
	LastName  string `json:"lastName"`
}

type authResponse struct {
	Token     string       `json:"token"`
	ExpiresAt time.Time    `json:"expiresAt"`
	User      userResponse `json:"user"`
}

func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	req, err := httpx.DecodeJSON[registerRequest](r)
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "Bad request")
		return
	}

	result, err := h.users.Register(ctx, users.RegisterInput{
		Email:     req.Email,
		Username:  req.Username,
		FirstName: req.FirstName,
		LastName:  req.LastName,
		Password:  req.Password,
	})

	if err != nil {
		switch {
		case errors.Is(err, users.ErrValidation):
			httpx.WriteError(w, http.StatusBadRequest, err.Error())
		case errors.Is(err, users.ErrConflict):
			httpx.WriteError(w, http.StatusConflict, err.Error())
		default:
			httpx.WriteError(w, http.StatusInternalServerError, err.Error())
		}
	}

	httpx.WriteJSON(w, http.StatusCreated, authResponse{
		Token:     result.Token,
		ExpiresAt: result.ExpiresAt,
		User: userResponse{
			ID:        result.User.ID,
			Email:     result.User.Email,
			Username:  result.User.Username,
			FirstName: result.User.FirstName,
			LastName:  result.User.LastName,
		},
	})

}

func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	req, err := httpx.DecodeJSON[loginRequest](r)
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "Bad request")
		return
	}

	result, err := h.users.Login(ctx, users.LoginInput{Identifier: req.Identifier, Password: req.Password})
	if err != nil {
		switch {
		case errors.Is(err, users.ErrValidation):
			httpx.WriteError(w, http.StatusBadRequest, err.Error())
		case errors.Is(err, users.ErrUnauthorized):
			httpx.WriteError(w, http.StatusUnauthorized, err.Error())
		default:
			httpx.WriteError(w, http.StatusInternalServerError, err.Error())
		}
		return
	}

	httpx.WriteJSON(w, http.StatusOK, authResponse{
		Token:     result.Token,
		ExpiresAt: result.ExpiresAt,
		User: userResponse{
			ID:        result.User.ID,
			Email:     result.User.Email,
			Username:  result.User.Username,
			FirstName: result.User.FirstName,
			LastName:  result.User.LastName,
		},
	})
}

func (h *AuthHandler) Me(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	userID, ok := middleware.UserIDFromContext(ctx)
	if !ok {
		httpx.WriteError(w, http.StatusUnauthorized, "unauthenticated")
		return
	}

	result, err := h.users.Me(ctx, userID)

	if err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	httpx.WriteJSON(w, http.StatusOK, userResponse{
		ID:        result.ID,
		Email:     result.Email,
		Username:  result.Username,
		FirstName: result.FirstName,
		LastName:  result.LastName,
	})
}
