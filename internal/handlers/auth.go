package handlers

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/mail"
	"strings"
	"time"

	"gorm.io/gorm"

	"github.com/alekseikl/additizer-api/internal/auth"
	gen "github.com/alekseikl/additizer-api/internal/generated"
	"github.com/alekseikl/additizer-api/internal/httpx"
	"github.com/alekseikl/additizer-api/internal/middleware"
	"github.com/alekseikl/additizer-api/internal/models"
	"github.com/google/uuid"
)

// "$(go env GOPATH)/bin/gorm" gen -i ./internal/models -o ./internal/generated
type AuthHandler struct {
	db         *gorm.DB
	issuer     *auth.TokenIssuer
	bcryptCost int
}

func NewAuthHandler(db *gorm.DB, issuer *auth.TokenIssuer, bcryptCost int) *AuthHandler {
	return &AuthHandler{db: db, issuer: issuer, bcryptCost: bcryptCost}
}

type registerRequest struct {
	Email    string `json:"email"`
	Username string `json:"username"`
	Password string `json:"password"`
}

type loginRequest struct {
	Identifier string `json:"identifier"`
	Password   string `json:"password"`
}

type userResponse struct {
	ID        string    `json:"id"`
	Email     string    `json:"email"`
	Username  string    `json:"username"`
	CreatedAt time.Time `json:"created_at"`
}

type authResponse struct {
	Token     string       `json:"token"`
	ExpiresAt time.Time    `json:"expires_at"`
	User      userResponse `json:"user"`
}

func toUserResponse(u *models.User) userResponse {
	return userResponse{
		ID:        u.ID.String(),
		Email:     u.Email,
		Username:  u.Username,
		CreatedAt: u.CreatedAt,
	}
}

func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	req, err := decodeJSON[registerRequest](r)
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "Bad request")
		return
	}

	req.Email = strings.TrimSpace(strings.ToLower(req.Email))
	req.Username = strings.TrimSpace(req.Username)

	if err := validateRegister(req); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}

	hash, err := auth.HashPassword(req.Password, h.bcryptCost)
	if err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, "Internal error")
		return
	}

	count, err := gorm.G[models.User](h.db).
		Where(gen.User.Username.Eq(req.Username)).
		Or(gen.User.Email.Eq(req.Email)).
		Count(ctx, gen.User.ID.Column().Name)

	if err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, "Internal error")
		return
	}

	if count > 0 {
		httpx.WriteError(w, http.StatusConflict, "Email or username already in use")
		return
	}

	user := models.User{
		ID:           uuid.New(),
		Email:        req.Email,
		Username:     req.Username,
		PasswordHash: hash,
	}

	if err := gorm.G[models.User](h.db).Create(ctx, &user); err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, "Could not create user")
		return
	}

	token, expiresAt, err := h.issuer.Generate(user.ID, user.Email)
	if err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, "could not issue token")
		return
	}

	httpx.WriteJSON(w, http.StatusCreated, authResponse{
		Token:     token,
		ExpiresAt: expiresAt,
		User:      toUserResponse(&user),
	})
}

func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	req, err := decodeJSON[loginRequest](r)
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "Bad request")
		return
	}

	req.Identifier = strings.TrimSpace(req.Identifier)

	if req.Identifier == "" || req.Password == "" {
		httpx.WriteError(w, http.StatusBadRequest, "identifier and password are required")
		return
	}

	var q gorm.ChainInterface[models.User]

	if _, err := mail.ParseAddress(req.Identifier); err == nil {
		q = gorm.G[models.User](h.db).Where(gen.User.Email.Eq(req.Identifier))
	} else {
		q = gorm.G[models.User](h.db).Where(gen.User.Username.Eq(req.Identifier))
	}

	user, err := q.First(ctx)

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			httpx.WriteError(w, http.StatusUnauthorized, "invalid credentials")
			return
		}
		httpx.WriteError(w, http.StatusInternalServerError, "database error")
		return
	}

	if !auth.CheckPassword(user.PasswordHash, req.Password) {
		httpx.WriteError(w, http.StatusUnauthorized, "invalid credentials")
		return
	}

	token, expiresAt, err := h.issuer.Generate(user.ID, user.Email)
	if err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, "could not issue token")
		return
	}

	httpx.WriteJSON(w, http.StatusOK, authResponse{
		Token:     token,
		ExpiresAt: expiresAt,
		User:      toUserResponse(&user),
	})
}

func (h *AuthHandler) Me(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	userID, ok := middleware.UserIDFromContext(ctx)
	if !ok {
		httpx.WriteError(w, http.StatusUnauthorized, "unauthenticated")
		return
	}

	user, err := gorm.G[models.User](h.db).Where(gen.User.ID.Eq(userID)).First(ctx)

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			httpx.WriteError(w, http.StatusNotFound, "user not found")
			return
		}
		httpx.WriteError(w, http.StatusInternalServerError, "database error")
		return
	}

	httpx.WriteJSON(w, http.StatusOK, toUserResponse(&user))
}

func decodeJSON[T any](r *http.Request) (T, error) {
	var v T
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(&v); err != nil {
		return v, errors.New("invalid json body")
	}
	return v, nil
}

func validateRegister(req registerRequest) error {
	if req.Email == "" || req.Username == "" || req.Password == "" {
		return errors.New("email, username, and password are required")
	}
	if _, err := mail.ParseAddress(req.Email); err != nil {
		return errors.New("invalid email address")
	}
	if len(req.Username) < 3 || len(req.Username) > 64 {
		return errors.New("username must be between 3 and 64 characters")
	}
	if len(req.Password) < 8 {
		return errors.New("password must be at least 8 characters")
	}
	return nil
}
