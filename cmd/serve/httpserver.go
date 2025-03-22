package serve

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"

	"github.com/computer-technology-team/go-judge/config"
	"github.com/computer-technology-team/go-judge/internal/auth"
	"github.com/computer-technology-team/go-judge/internal/home"
	"github.com/computer-technology-team/go-judge/internal/problems"
	"github.com/computer-technology-team/go-judge/internal/profiles"
	"github.com/computer-technology-team/go-judge/internal/submissions"
	"github.com/computer-technology-team/go-judge/web/static"
	"github.com/computer-technology-team/go-judge/web/templates"
)

// StartServer sets up and starts the HTTP server
func StartServer(cfg config.ServerConfig) error {
	// Setup structured logging
	logHandler := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	})
	slog.SetDefault(slog.New(logHandler))

	homeHandler, err := createHomeHandler()
	if err != nil {
		return fmt.Errorf("could not create home handler: %w", err)
	}

	// Create a new router
	router := chi.NewRouter()

	// Middleware
	router.Use(middleware.Logger)
	router.Use(middleware.Recoverer)
	router.Use(middleware.RealIP)
	router.Use(middleware.RequestID)
	router.Use(middleware.Timeout(60 * time.Second))

	// CORS configuration
	router.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"*"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token"},
		ExposedHeaders:   []string{"Link"},
		AllowCredentials: true,
		MaxAge:           300,
	}))

	// API routes
	router.Route("/", func(r chi.Router) {
		// Auth routes
		r.Route("/auth", auth.NewRoutes(auth.NewHandler()))

		// Problem routes
		r.Route("/problems", problems.NewRoutes(problems.NewHandler()))

		// Submission routes
		r.Route("/submissions", submissions.NewRoutes(submissions.NewHandler()))

		// Profile routes
		r.Route("/profiles", profiles.NewRoutes(profiles.NewHandler()))

		// Home routes
		r.Route("/", home.NewRoutes(homeHandler))
	})

	router.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	// Serve static files from embedded filesystem
	router.Handle("/static/*", http.StripPrefix("/static", static.StaticFilerHandler()))

	// Create the HTTP server
	server := &http.Server{
		Addr:         fmt.Sprintf("%s:%d", cfg.Host, cfg.Port),
		Handler:      router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Start server in a goroutine
	go func() {
		slog.Info("Server starting", "address", server.Addr)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("Server failed to start", "error", err)
			os.Exit(1)
		}
	}()

	// Wait for interrupt signal to gracefully shutdown the server
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	slog.Info("Server is shutting down")

	// Create a deadline to wait for
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Doesn't block if no connections, but will otherwise wait
	// until the timeout deadline
	if err := server.Shutdown(ctx); err != nil {
		slog.Error("Server forced to shutdown", "error", err)
		return err
	}

	slog.Info("Server exited properly")
	return nil
}

func createHomeHandler() (home.Handler, error) {
	tmpls, err := templates.GetTemplates(templates.Home)
	if err != nil {
		return nil, fmt.Errorf("could not get templates: %w", err)
	}

	return home.NewHandler(tmpls), nil
}
