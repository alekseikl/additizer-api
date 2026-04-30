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
	ErrNotFound     = errors.New("not found")
)

type RegisterInput struct {
	Email     string
	Username  string
	FirstName string
	LastName  string
	Password  string
}

func (r *RegisterInput) normalize() {
	r.Email = strings.TrimSpace(strings.ToLower(r.Email))
	r.Username = strings.TrimSpace(r.Username)
	r.FirstName = strings.TrimSpace(r.FirstName)
	r.LastName = strings.TrimSpace(r.LastName)
}

func (r *RegisterInput) validate() error {
	if r.Email == "" || r.Username == "" || r.FirstName == "" || r.LastName == "" || r.Password == "" {
		return fmt.Errorf("%w: email, username, first name, last name, and password are required", ErrValidation)
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

type UpdateUserInput struct {
	Email     *string
	Username  *string
	FirstName *string
	LastName  *string
}

func (u *UpdateUserInput) normalize() {
	if u.Email != nil {
		v := strings.TrimSpace(strings.ToLower(*u.Email))
		u.Email = &v
	}
	if u.Username != nil {
		v := strings.TrimSpace(*u.Username)
		u.Username = &v
	}
	if u.FirstName != nil {
		v := strings.TrimSpace(*u.FirstName)
		u.FirstName = &v
	}
	if u.LastName != nil {
		v := strings.TrimSpace(*u.LastName)
		u.LastName = &v
	}
}

func (u *UpdateUserInput) validate() error {
	if u.Email == nil && u.Username == nil && u.FirstName == nil && u.LastName == nil {
		return fmt.Errorf("%w: at least one field must be provided", ErrValidation)
	}
	if u.Email != nil {
		if *u.Email == "" {
			return fmt.Errorf("%w: email cannot be empty", ErrValidation)
		}
		if _, err := mail.ParseAddress(*u.Email); err != nil {
			return fmt.Errorf("%w: invalid email address", ErrValidation)
		}
	}
	if u.Username != nil {
		if len(*u.Username) < 3 || len(*u.Username) > 64 {
			return fmt.Errorf("%w: username must be between 3 and 64 characters", ErrValidation)
		}
	}
	if u.FirstName != nil && *u.FirstName == "" {
		return fmt.Errorf("%w: first name cannot be empty", ErrValidation)
	}
	if u.LastName != nil && *u.LastName == "" {
		return fmt.Errorf("%w: last name cannot be empty", ErrValidation)
	}
	return nil
}

type UserResult struct {
	ID        string
	Email     string
	Username  string
	FirstName string
	LastName  string
}

type AuthResult struct {
	Token     string
	ExpiresAt time.Time
	User      UserResult
}
