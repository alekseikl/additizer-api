package users

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/alekseikl/additizer-api/internal/auth"
	"github.com/alekseikl/additizer-api/internal/config"
	"github.com/alekseikl/additizer-api/internal/models"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func newTestService(t *testing.T) (*Service, *gorm.DB) {
	t.Helper()

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatalf("open test database: %v", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("get sql database: %v", err)
	}
	sqlDB.SetMaxOpenConns(1)
	t.Cleanup(func() {
		if err := sqlDB.Close(); err != nil {
			t.Fatalf("close test database: %v", err)
		}
	})

	if err := db.AutoMigrate(&models.User{}); err != nil {
		t.Fatalf("migrate test database: %v", err)
	}

	cfg := &config.Config{
		JWTSecret:     []byte("unit-test-secret"),
		JWTExpiration: time.Hour,
		BcryptCost:    bcrypt.MinCost,
	}

	return NewService(db, cfg), db
}

func TestServiceRegisterCreatesUserAndToken(t *testing.T) {
	svc, db := newTestService(t)
	ctx := context.Background()

	before := time.Now()
	result, err := svc.Register(ctx, RegisterInput{
		Email:     " Alice@Example.COM ",
		Username:  " alice ",
		FirstName: "Alice",
		LastName:  "Anderson",
		Password:  "secret-password",
	})
	if err != nil {
		t.Fatalf("register user: %v", err)
	}

	if result.Token == "" {
		t.Fatal("expected token")
	}
	if result.ExpiresAt.Before(before) {
		t.Fatalf("expected future expiration, got %s", result.ExpiresAt)
	}
	if result.User.Email != "alice@example.com" {
		t.Fatalf("expected normalized email, got %q", result.User.Email)
	}
	if result.User.Username != "alice" {
		t.Fatalf("expected trimmed username, got %q", result.User.Username)
	}

	claims, err := svc.Issuer().Parse(result.Token)
	if err != nil {
		t.Fatalf("parse issued token: %v", err)
	}
	if claims.UserID.String() != result.User.ID {
		t.Fatalf("expected token user ID %q, got %q", result.User.ID, claims.UserID)
	}
	if claims.Email != result.User.Email {
		t.Fatalf("expected token email %q, got %q", result.User.Email, claims.Email)
	}

	var stored models.User
	if err := db.First(&stored, "id = ?", result.User.ID).Error; err != nil {
		t.Fatalf("load stored user: %v", err)
	}
	if stored.Email != "alice@example.com" || stored.Username != "alice" {
		t.Fatalf("stored user was not normalized: %#v", stored)
	}
	if stored.PasswordHash == "secret-password" {
		t.Fatal("stored password should be hashed")
	}
	if !auth.CheckPassword(stored.PasswordHash, "secret-password") {
		t.Fatal("stored password hash does not match password")
	}
}

func TestServiceRegisterRejectsInvalidInput(t *testing.T) {
	svc, _ := newTestService(t)
	ctx := context.Background()

	tests := []struct {
		name  string
		input RegisterInput
	}{
		{
			name: "missing email",
			input: RegisterInput{
				Username:  "alice",
				FirstName: "Alice",
				LastName:  "Anderson",
				Password:  "secret-password",
			},
		},
		{
			name: "invalid email",
			input: RegisterInput{
				Email:     "not-an-email",
				Username:  "alice",
				FirstName: "Alice",
				LastName:  "Anderson",
				Password:  "secret-password",
			},
		},
		{
			name: "short username",
			input: RegisterInput{
				Email:     "alice@example.com",
				Username:  "al",
				FirstName: "Alice",
				LastName:  "Anderson",
				Password:  "secret-password",
			},
		},
		{
			name: "short password",
			input: RegisterInput{
				Email:     "alice@example.com",
				Username:  "alice",
				FirstName: "Alice",
				LastName:  "Anderson",
				Password:  "short",
			},
		},
		{
			name: "missing first name",
			input: RegisterInput{
				Email:    "alice@example.com",
				Username: "alice",
				LastName: "Anderson",
				Password: "secret-password",
			},
		},
		{
			name: "missing last name",
			input: RegisterInput{
				Email:     "alice@example.com",
				Username:  "alice",
				FirstName: "Alice",
				Password:  "secret-password",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := svc.Register(ctx, tt.input)
			if result != nil {
				t.Fatalf("expected no result, got %#v", result)
			}
			if !errors.Is(err, ErrValidation) {
				t.Fatalf("expected validation error, got %v", err)
			}
		})
	}
}

func TestServiceRegisterRejectsDuplicateEmailOrUsername(t *testing.T) {
	svc, _ := newTestService(t)
	ctx := context.Background()

	_, err := svc.Register(ctx, RegisterInput{
		Email:     "alice@example.com",
		Username:  "alice",
		FirstName: "Alice",
		LastName:  "Anderson",
		Password:  "secret-password",
	})
	if err != nil {
		t.Fatalf("register first user: %v", err)
	}

	result, err := svc.Register(ctx, RegisterInput{
		Email:     "alice@example.com",
		Username:  "alice2",
		FirstName: "Alice",
		LastName:  "Anderson",
		Password:  "secret-password",
	})
	if result != nil {
		t.Fatalf("expected no result, got %#v", result)
	}
	if !errors.Is(err, ErrConflict) {
		t.Fatalf("expected conflict error, got %v", err)
	}

	result, err = svc.Register(ctx, RegisterInput{
		Email:     "alice2@example.com",
		Username:  "alice",
		FirstName: "Alice",
		LastName:  "Anderson",
		Password:  "secret-password",
	})
	if result != nil {
		t.Fatalf("expected no result, got %#v", result)
	}
	if !errors.Is(err, ErrConflict) {
		t.Fatalf("expected conflict error, got %v", err)
	}
}

func TestServiceLoginIssuesTokenForEmailOrUsername(t *testing.T) {
	svc, _ := newTestService(t)
	ctx := context.Background()

	registered, err := svc.Register(ctx, RegisterInput{
		Email:     "alice@example.com",
		Username:  "alice",
		FirstName: "Alice",
		LastName:  "Anderson",
		Password:  "secret-password",
	})
	if err != nil {
		t.Fatalf("register user: %v", err)
	}

	tests := []struct {
		name       string
		identifier string
	}{
		{name: "email", identifier: " alice@example.com "},
		{name: "username", identifier: " alice "},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := svc.Login(ctx, LoginInput{
				Identifier: tt.identifier,
				Password:   "secret-password",
			})
			if err != nil {
				t.Fatalf("login: %v", err)
			}
			if result.Token == "" {
				t.Fatal("expected token")
			}
			if result.User != registered.User {
				t.Fatalf("expected registered user %#v, got %#v", registered.User, result.User)
			}
		})
	}
}

func TestServiceLoginRejectsInvalidCredentials(t *testing.T) {
	svc, _ := newTestService(t)
	ctx := context.Background()

	_, err := svc.Register(ctx, RegisterInput{
		Email:     "alice@example.com",
		Username:  "alice",
		FirstName: "Alice",
		LastName:  "Anderson",
		Password:  "secret-password",
	})
	if err != nil {
		t.Fatalf("register user: %v", err)
	}

	tests := []struct {
		name  string
		input LoginInput
		want  error
	}{
		{
			name:  "missing identifier",
			input: LoginInput{Password: "secret-password"},
			want:  ErrValidation,
		},
		{
			name: "wrong password",
			input: LoginInput{
				Identifier: "alice",
				Password:   "wrong-password",
			},
			want: ErrUnauthorized,
		},
		{
			name: "unknown user",
			input: LoginInput{
				Identifier: "bob",
				Password:   "secret-password",
			},
			want: ErrUnauthorized,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := svc.Login(ctx, tt.input)
			if result != nil {
				t.Fatalf("expected no result, got %#v", result)
			}
			if !errors.Is(err, tt.want) {
				t.Fatalf("expected %v, got %v", tt.want, err)
			}
		})
	}
}

func TestServiceUpdateChangesProvidedFields(t *testing.T) {
	svc, db := newTestService(t)
	ctx := context.Background()

	registered, err := svc.Register(ctx, RegisterInput{
		Email:     "alice@example.com",
		Username:  "alice",
		FirstName: "Alice",
		LastName:  "Anderson",
		Password:  "secret-password",
	})
	if err != nil {
		t.Fatalf("register user: %v", err)
	}

	userID, err := uuid.Parse(registered.User.ID)
	if err != nil {
		t.Fatalf("parse user ID: %v", err)
	}

	result, err := svc.Update(ctx, userID, UpdateUserInput{
		Email:     new(" New@Example.COM "),
		Username:  new(" new_alice "),
		FirstName: new(" Alicia "),
		LastName:  new(" Brown "),
	})
	if err != nil {
		t.Fatalf("update user: %v", err)
	}

	if result.Email != "new@example.com" {
		t.Fatalf("expected normalized email, got %q", result.Email)
	}
	if result.Username != "new_alice" {
		t.Fatalf("expected trimmed username, got %q", result.Username)
	}
	if result.FirstName != "Alicia" {
		t.Fatalf("expected trimmed first name, got %q", result.FirstName)
	}
	if result.LastName != "Brown" {
		t.Fatalf("expected trimmed last name, got %q", result.LastName)
	}
	if result.ID != registered.User.ID {
		t.Fatalf("expected same user ID %q, got %q", registered.User.ID, result.ID)
	}

	var stored models.User
	if err := db.First(&stored, "id = ?", registered.User.ID).Error; err != nil {
		t.Fatalf("load stored user: %v", err)
	}
	if stored.Email != "new@example.com" || stored.Username != "new_alice" ||
		stored.FirstName != "Alicia" || stored.LastName != "Brown" {
		t.Fatalf("stored user not updated: %#v", stored)
	}
}

func TestServiceUpdatePartialFieldsLeavesOthersUnchanged(t *testing.T) {
	svc, _ := newTestService(t)
	ctx := context.Background()

	registered, err := svc.Register(ctx, RegisterInput{
		Email:     "alice@example.com",
		Username:  "alice",
		FirstName: "Alice",
		LastName:  "Anderson",
		Password:  "secret-password",
	})
	if err != nil {
		t.Fatalf("register user: %v", err)
	}

	userID, err := uuid.Parse(registered.User.ID)
	if err != nil {
		t.Fatalf("parse user ID: %v", err)
	}

	result, err := svc.Update(ctx, userID, UpdateUserInput{
		FirstName: new("Alicia"),
	})
	if err != nil {
		t.Fatalf("update user: %v", err)
	}

	if result.FirstName != "Alicia" {
		t.Fatalf("expected updated first name, got %q", result.FirstName)
	}
	if result.Email != registered.User.Email {
		t.Fatalf("expected unchanged email %q, got %q", registered.User.Email, result.Email)
	}
	if result.Username != registered.User.Username {
		t.Fatalf("expected unchanged username %q, got %q", registered.User.Username, result.Username)
	}
	if result.LastName != registered.User.LastName {
		t.Fatalf("expected unchanged last name %q, got %q", registered.User.LastName, result.LastName)
	}
}

func TestServiceUpdateAllowsKeepingOwnEmailAndUsername(t *testing.T) {
	svc, _ := newTestService(t)
	ctx := context.Background()

	registered, err := svc.Register(ctx, RegisterInput{
		Email:     "alice@example.com",
		Username:  "alice",
		FirstName: "Alice",
		LastName:  "Anderson",
		Password:  "secret-password",
	})
	if err != nil {
		t.Fatalf("register user: %v", err)
	}

	userID, err := uuid.Parse(registered.User.ID)
	if err != nil {
		t.Fatalf("parse user ID: %v", err)
	}

	if _, err := svc.Update(ctx, userID, UpdateUserInput{
		Email:    new("alice@example.com"),
		Username: new("alice"),
	}); err != nil {
		t.Fatalf("update user with own credentials: %v", err)
	}
}

func TestServiceUpdateRejectsInvalidInput(t *testing.T) {
	svc, _ := newTestService(t)
	ctx := context.Background()

	registered, err := svc.Register(ctx, RegisterInput{
		Email:     "alice@example.com",
		Username:  "alice",
		FirstName: "Alice",
		LastName:  "Anderson",
		Password:  "secret-password",
	})
	if err != nil {
		t.Fatalf("register user: %v", err)
	}

	userID, err := uuid.Parse(registered.User.ID)
	if err != nil {
		t.Fatalf("parse user ID: %v", err)
	}

	tests := []struct {
		name  string
		input UpdateUserInput
	}{
		{name: "no fields", input: UpdateUserInput{}},
		{name: "empty email", input: UpdateUserInput{Email: new("   ")}},
		{name: "invalid email", input: UpdateUserInput{Email: new("not-an-email")}},
		{name: "short username", input: UpdateUserInput{Username: new("al")}},
		{name: "empty first name", input: UpdateUserInput{FirstName: new("  ")}},
		{name: "empty last name", input: UpdateUserInput{LastName: new("")}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := svc.Update(ctx, userID, tt.input)
			if result != nil {
				t.Fatalf("expected no result, got %#v", result)
			}
			if !errors.Is(err, ErrValidation) {
				t.Fatalf("expected validation error, got %v", err)
			}
		})
	}
}

func TestServiceUpdateRejectsConflictingEmailOrUsername(t *testing.T) {
	svc, _ := newTestService(t)
	ctx := context.Background()

	if _, err := svc.Register(ctx, RegisterInput{
		Email:     "alice@example.com",
		Username:  "alice",
		FirstName: "Alice",
		LastName:  "Anderson",
		Password:  "secret-password",
	}); err != nil {
		t.Fatalf("register first user: %v", err)
	}

	registered, err := svc.Register(ctx, RegisterInput{
		Email:     "bob@example.com",
		Username:  "bob",
		FirstName: "Bob",
		LastName:  "Brown",
		Password:  "secret-password",
	})
	if err != nil {
		t.Fatalf("register second user: %v", err)
	}

	userID, err := uuid.Parse(registered.User.ID)
	if err != nil {
		t.Fatalf("parse user ID: %v", err)
	}

	tests := []struct {
		name  string
		input UpdateUserInput
	}{
		{name: "email", input: UpdateUserInput{Email: new("alice@example.com")}},
		{name: "username", input: UpdateUserInput{Username: new("alice")}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := svc.Update(ctx, userID, tt.input)
			if result != nil {
				t.Fatalf("expected no result, got %#v", result)
			}
			if !errors.Is(err, ErrConflict) {
				t.Fatalf("expected conflict error, got %v", err)
			}
		})
	}
}

func TestServiceUpdateUnknownUserReturnsNotFound(t *testing.T) {
	svc, _ := newTestService(t)
	ctx := context.Background()

	result, err := svc.Update(ctx, uuid.New(), UpdateUserInput{
		FirstName: new("Ghost"),
	})
	if result != nil {
		t.Fatalf("expected no result, got %#v", result)
	}
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected not found error, got %v", err)
	}
}

func TestServiceMeReturnsCurrentUser(t *testing.T) {
	svc, _ := newTestService(t)
	ctx := context.Background()

	registered, err := svc.Register(ctx, RegisterInput{
		Email:     "alice@example.com",
		Username:  "alice",
		FirstName: "Alice",
		LastName:  "Anderson",
		Password:  "secret-password",
	})
	if err != nil {
		t.Fatalf("register user: %v", err)
	}

	userID, err := uuid.Parse(registered.User.ID)
	if err != nil {
		t.Fatalf("parse user ID: %v", err)
	}

	result, err := svc.Me(ctx, userID)
	if err != nil {
		t.Fatalf("load current user: %v", err)
	}
	if *result != registered.User {
		t.Fatalf("expected registered user %#v, got %#v", registered.User, *result)
	}
}
