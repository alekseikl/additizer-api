package server

import (
	"net/http"

	"github.com/alekseikl/additizer-api/internal/handlers"
	"github.com/alekseikl/additizer-api/internal/middleware"
	"github.com/alekseikl/additizer-api/internal/presets"
	"github.com/alekseikl/additizer-api/internal/users"
)

type Deps struct {
	Users   *users.Service
	Presets *presets.Service
}

type Server struct {
	auth        *handlers.AuthHandler
	presets     *handlers.PresetsHandler
	requireAuth func(http.Handler) http.Handler

	handler http.Handler
}

func New(deps Deps) *Server {
	s := &Server{
		auth:        handlers.NewAuthHandler(deps.Users),
		presets:     handlers.NewPresetsHandler(deps.Presets),
		requireAuth: middleware.RequireAuth(deps.Users.Issuer()),
	}
	s.handler = s.routes()
	return s
}

func (s *Server) Handler() http.Handler {
	return s.handler
}
