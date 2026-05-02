package server

import (
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"

	"github.com/alekseikl/additizer-api/internal/handlers"
)

func (s *Server) routes() http.Handler {
	r := chi.NewRouter()

	r.Use(chimw.RequestID)
	r.Use(chimw.RealIP)
	r.Use(chimw.Logger)
	r.Use(chimw.Recoverer)
	r.Use(chimw.Timeout(30 * time.Second))

	r.Get("/healthz", handlers.Health)

	r.Route("/api", func(r chi.Router) {
		r.Route("/auth", func(r chi.Router) {
			r.Post("/register", s.auth.Register)
			r.Post("/login", s.auth.Login)
		})

		r.Group(func(r chi.Router) {
			r.Use(s.requireAuth)

			r.Get("/me", s.auth.Me)

			r.Route("/presets", func(r chi.Router) {
				r.Get("/", s.presets.ListPresets)
				r.Post("/", s.presets.CreatePreset)
				r.Put("/{presetID}", s.presets.UpdatePreset)
				r.Delete("/{presetID}", s.presets.DeletePreset)
			})

			r.Route("/groups", func(r chi.Router) {
				r.Get("/", s.presets.ListGroups)
				r.Post("/", s.presets.CreateGroup)
				r.Put("/{groupID}", s.presets.UpdateGroup)
				r.Delete("/{groupID}", s.presets.DeleteGroup)
				r.Get("/{groupID}/presets", s.presets.ListPresetsInGroup)
			})
		})
	})

	return r
}
