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
	"github.com/alekseikl/additizer-api/internal/httpx"
	"github.com/alekseikl/additizer-api/internal/middleware"
	"github.com/alekseikl/additizer-api/internal/models"
)

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
	var req registerRequest
	if err := decodeJSON(r, &req); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, err.Error())
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
		httpx.WriteError(w, http.StatusInternalServerError, "could not hash password")
		return
	}

	user := models.User{
		Email:        req.Email,
		Username:     req.Username,
		PasswordHash: hash,
	}

	if err := h.db.WithContext(r.Context()).Create(&user).Error; err != nil {
		if isUniqueViolation(err) {
			httpx.WriteError(w, http.StatusConflict, "email or username already in use")
			return
		}
		httpx.WriteError(w, http.StatusInternalServerError, "could not create user")
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
	var req loginRequest
	if err := decodeJSON(r, &req); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}

	req.Identifier = strings.TrimSpace(req.Identifier)
	if req.Identifier == "" || req.Password == "" {
		httpx.WriteError(w, http.StatusBadRequest, "identifier and password are required")
		return
	}

	var user models.User
	query := h.db.WithContext(r.Context())
	if _, err := mail.ParseAddress(req.Identifier); err == nil {
		query = query.Where("email = ?", strings.ToLower(req.Identifier))
	} else {
		query = query.Where("username = ?", req.Identifier)
	}

	if err := query.First(&user).Error; err != nil {
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
	userID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		httpx.WriteError(w, http.StatusUnauthorized, "unauthenticated")
		return
	}

	var user models.User
	if err := h.db.WithContext(r.Context()).First(&user, "id = ?", userID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			httpx.WriteError(w, http.StatusNotFound, "user not found")
			return
		}
		httpx.WriteError(w, http.StatusInternalServerError, "database error")
		return
	}

	httpx.WriteJSON(w, http.StatusOK, toUserResponse(&user))
}

func decodeJSON(r *http.Request, v any) error {
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(v); err != nil {
		return errors.New("invalid json body")
	}
	return nil
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

func isUniqueViolation(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "sqlstate 23505") ||
		strings.Contains(msg, "duplicate key") ||
		strings.Contains(msg, "unique constraint")
}
