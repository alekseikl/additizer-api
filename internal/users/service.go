package users

import (
	"context"
	"errors"
	"fmt"
	"net/mail"

	"github.com/alekseikl/additizer-api/internal/auth"
	"github.com/alekseikl/additizer-api/internal/config"
	g "github.com/alekseikl/additizer-api/internal/generated"
	"github.com/alekseikl/additizer-api/internal/models"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Service struct {
	db         *gorm.DB
	issuer     *auth.TokenIssuer
	bcryptCost int
}

func NewService(db *gorm.DB, cfg *config.Config) *Service {
	issuer := auth.NewTokenIssuer(cfg.JWTSecret, cfg.JWTExpiration)

	return &Service{db: db, issuer: issuer, bcryptCost: cfg.BcryptCost}
}

func (s *Service) Issuer() *auth.TokenIssuer {
	return s.issuer
}

func (s *Service) Register(ctx context.Context, reg RegisterInput) (*AuthResult, error) {
	reg.normalize()

	if err := reg.validate(); err != nil {
		return nil, err
	}

	hash, err := auth.HashPassword(reg.Password, s.bcryptCost)

	if err != nil {
		return nil, ErrInternal
	}

	_, err = gorm.G[models.User](s.db).
		Select(g.User.ID.Column().Name).
		Where(g.User.Username.Eq(reg.Username)).
		Or(g.User.Email.Eq(reg.Email)).First(ctx)

	if err == nil {
		return nil, fmt.Errorf("%w: email or username already in use", ErrConflict)
	}

	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrInternal
	}

	user := models.User{
		ID:           uuid.New(),
		Email:        reg.Email,
		Username:     reg.Username,
		PasswordHash: hash,
	}

	if err := gorm.G[models.User](s.db).Create(ctx, &user); err != nil {
		return nil, ErrInternal
	}

	token, expiresAt, err := s.issuer.Generate(user.ID, user.Email)
	if err != nil {
		return nil, ErrInternal
	}

	return &AuthResult{Token: token, ExpiresAt: expiresAt, User: UserResult{
		ID:       user.ID.String(),
		Email:    user.Email,
		Username: user.Username,
	}}, nil
}

func (s *Service) Login(ctx context.Context, login LoginInput) (*AuthResult, error) {
	login.normalize()

	if err := login.validate(); err != nil {
		return nil, err
	}

	var q gorm.ChainInterface[models.User]

	if _, err := mail.ParseAddress(login.Identifier); err == nil {
		q = gorm.G[models.User](s.db).Where(g.User.Email.Eq(login.Identifier))
	} else {
		q = gorm.G[models.User](s.db).Where(g.User.Username.Eq(login.Identifier))
	}

	user, err := q.First(ctx)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrUnauthorized
		}
		return nil, ErrInternal
	}

	if !auth.CheckPassword(user.PasswordHash, login.Password) {
		return nil, ErrUnauthorized
	}

	token, expiresAt, err := s.issuer.Generate(user.ID, user.Email)
	if err != nil {
		return nil, ErrInternal
	}

	return &AuthResult{Token: token, ExpiresAt: expiresAt, User: UserResult{
		ID:       user.ID.String(),
		Email:    user.Email,
		Username: user.Username,
	}}, nil
}

func (s *Service) Me(ctx context.Context, userID uuid.UUID) (*UserResult, error) {
	user, err := gorm.G[models.User](s.db).Where(g.User.ID.Eq(userID)).First(ctx)

	if err != nil {
		return nil, ErrInternal
	}

	return &UserResult{
		ID:       user.ID.String(),
		Email:    user.Email,
		Username: user.Username,
	}, nil
}
