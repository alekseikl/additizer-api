package users

import (
	"errors"
	"fmt"
	"net/mail"
	"strings"
	"time"
)

var (
	ErrValidation   = errors.New("validation failed")
	ErrConflict     = errors.New("conflict")
	ErrInternal     = errors.New("internal")
	ErrUnauthorized = errors.New("unauthorized")
)

type RegisterInput struct {
	Email    string
	Username string
	Password string
}

func (r *RegisterInput) normalize() {
	r.Email = strings.TrimSpace(strings.ToLower(r.Email))
	r.Username = strings.TrimSpace(r.Username)
}

func (r *RegisterInput) validate() error {
	if r.Email == "" || r.Username == "" || r.Password == "" {
		return fmt.Errorf("%w: email, username, and password are required", ErrValidation)
	}
	if _, err := mail.ParseAddress(r.Email); err != nil {
		return fmt.Errorf("%w: invalid email address", ErrValidation)
	}
	if len(r.Username) < 3 || len(r.Username) > 64 {
		return fmt.Errorf("%w: username must be between 3 and 64 characters", ErrValidation)
	}
	if len(r.Password) < 8 {
		return fmt.Errorf("%w: password must be at least 8 characters", ErrValidation)
	}
	return nil
}

type LoginInput struct {
	Identifier string
	Password   string
}

func (l *LoginInput) normalize() {
	l.Identifier = strings.TrimSpace(l.Identifier)
}

func (l *LoginInput) validate() error {
	if l.Identifier == "" || l.Password == "" {
		return fmt.Errorf("%w: identifier and password are required", ErrValidation)

	}

	return nil
}

type UserResult struct {
	ID       string
	Email    string
	Username string
}

type AuthResult struct {
	Token     string
	ExpiresAt time.Time
	User      UserResult
}
