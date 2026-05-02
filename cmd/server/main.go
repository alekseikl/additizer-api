package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"

	"github.com/alekseikl/additizer-api/internal/config"
	"github.com/alekseikl/additizer-api/internal/database"
	"github.com/alekseikl/additizer-api/internal/handlers"
	"github.com/alekseikl/additizer-api/internal/middleware"
	"github.com/alekseikl/additizer-api/internal/presets"
	"github.com/alekseikl/additizer-api/internal/users"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("load config: %v", err)
	}

	db, err := database.Connect(cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("connect database: %v", err)
	}

	if err := database.Migrate(db); err != nil {
		log.Fatalf("migrate database: %v", err)
	}

	usersService := users.NewService(db, cfg)
	presetsService := presets.NewService(db)

	authHandler := handlers.NewAuthHandler(usersService)
	presetsHandler := handlers.NewPresetsHandler(presetsService)
	requireAuth := middleware.RequireAuth(usersService.Issuer())

	r := chi.NewRouter()
	r.Use(chimw.RequestID)
	r.Use(chimw.RealIP)
	r.Use(chimw.Logger)
	r.Use(chimw.Recoverer)
	r.Use(chimw.Timeout(30 * time.Second))

	r.Get("/healthz", handlers.Health)

	r.Route("/api/v1", func(r chi.Router) {
		r.Route("/auth", func(r chi.Router) {
			r.Post("/register", authHandler.Register)
			r.Post("/login", authHandler.Login)
		})

		r.Group(func(r chi.Router) {
			r.Use(requireAuth)
			r.Get("/me", authHandler.Me)
			r.Route("/presets", func(r chi.Router) {
				r.Get("/groups", presetsHandler.ListGroups)
				r.Post("/groups", presetsHandler.CreateGroup)
				r.Put("/groups/{groupID}", presetsHandler.UpdateGroup)
				r.Delete("/groups/{groupID}", presetsHandler.DeleteGroup)
				r.Get("/groups/{groupID}/presets", presetsHandler.ListPresetsInGroup)

				r.Get("/", presetsHandler.ListPresets)
				r.Post("/", presetsHandler.CreatePreset)
				r.Put("/{presetID}", presetsHandler.UpdatePreset)
				r.Delete("/{presetID}", presetsHandler.DeletePreset)
			})
		})
	})

	server := &http.Server{
		Addr:              cfg.HTTPAddr,
		Handler:           r,
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      15 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	go func() {
		log.Printf("http server listening on %s", cfg.HTTPAddr)
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("server error: %v", err)
		}
	}()

	// presetsService.Check(context.Background())

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop
	log.Println("shutting down")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := server.Shutdown(ctx); err != nil {
		log.Printf("shutdown: %v", err)
	}

	if sqlDB, err := db.DB(); err == nil {
		_ = sqlDB.Close()
	}

}
