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
	chiMiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/computer-technology-team/go-judge/config"
	"github.com/computer-technology-team/go-judge/internal/auth"
	authenticatorPkg "github.com/computer-technology-team/go-judge/internal/auth/authenticator"
	runnerClient "github.com/computer-technology-team/go-judge/internal/clients/runner"
	"github.com/computer-technology-team/go-judge/internal/home"
	"github.com/computer-technology-team/go-judge/internal/middleware"
	"github.com/computer-technology-team/go-judge/internal/problems"
	"github.com/computer-technology-team/go-judge/internal/profiles"
	"github.com/computer-technology-team/go-judge/internal/storage"
	"github.com/computer-technology-team/go-judge/internal/submissions"
	"github.com/computer-technology-team/go-judge/web/static"
	"github.com/computer-technology-team/go-judge/web/templates"
)

func StartServer(ctx context.Context, cfg config.Config) error {
	pool, err := storage.NewPgxPool(ctx, cfg.Database)
	if err != nil {
		return fmt.Errorf("could not create database pool: %w", err)
	}

	querier := storage.New()

	runnerClient, err := runnerClient.NewClient(ctx, cfg.RunnerClient)
	if err != nil {
		return fmt.Errorf("could not create runner client: %w", err)
	}

	broker := submissions.NewBroker(cfg.Broker, runnerClient, querier, pool)

	broker.StartWorkers(ctx)
	defer broker.StopWorkers()

	authenticator, err := authenticatorPkg.NewAuthenticator(cfg.Authentication)
	if err != nil {
		return fmt.Errorf("could not create authenticator: %w", err)
	}

	homeHandler, err := createHomeHandler()
	if err != nil {
		return fmt.Errorf("could not create home handler: %w", err)
	}

	authServicer, err := createAuthenticationServicer(authenticator, pool, querier)
	if err != nil {
		return fmt.Errorf("could not create authenticantion servicer: %w", err)
	}

	profilesServicer, err := createProfilesServicer(pool, querier)
	if err != nil {
		return fmt.Errorf("could not create profiles servicer: %w", err)
	}

	problemTemplates, err := templates.GetTemplates(templates.Problems)
	if err != nil {
		return fmt.Errorf("could not get submit problem templates: %w", err)
	}

	submissionsServicer, err := createSubmissionsServicer(broker, pool, querier)
	if err != nil {
		return fmt.Errorf("could not create submission servicer: %w", err)
	}

	sharedTemplates, err := templates.GetSharedTemplates()
	if err != nil {
		return fmt.Errorf("could not get shared templates: %w", err)

	}

	// Create a new router
	router := chi.NewRouter()

	// Middleware
	router.Use(chiMiddleware.Logger)
	router.Use(middleware.NewRecoveryHandler(sharedTemplates))
	router.Use(chiMiddleware.RealIP)
	router.Use(chiMiddleware.RequestID)
	router.Use(chiMiddleware.Timeout(60 * time.Second))
	router.Use(middleware.NewAuthMiddleWare(authenticator, pool, querier, sharedTemplates))

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
		r.Route("/auth", auth.NewRoutes(authServicer, sharedTemplates))

		// Problem routes
		r.Route("/problems", problems.NewRoutes(problems.NewHandler(problemTemplates, pool, querier), sharedTemplates))

		// Submission routes
		r.With(middleware.NewRequireAuthMiddleware(sharedTemplates)).
			Route("/submissions", submissions.NewRoutes(submissionsServicer))

		// Profile routes
		r.Route("/profiles", profiles.NewRoutes(profilesServicer, sharedTemplates))

		// Home routes
		r.Route("/", home.NewRoutes(homeHandler))
	})

	router.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	// Serve static files from embedded filesystem with caching and ETag support
	router.Handle("/static/*", http.StripPrefix("/static", static.StaticFilerHandler()))

	// Create the HTTP server
	server := &http.Server{
		Addr:         fmt.Sprintf("%s:%d", cfg.JudgeServer.Host, cfg.JudgeServer.Port),
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
	var cancel func()
	ctx, cancel = context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	// Doesn't block if no connections, but will otherwise wait
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
		return nil, fmt.Errorf("could not get home templates: %w", err)
	}

	return home.NewHandler(tmpls), nil
}

func createProfilesServicer(pool *pgxpool.Pool, querier storage.Querier) (profiles.Servicer, error) {
	tmpls, err := templates.GetTemplates(templates.Profiles)
	if err != nil {
		return nil, fmt.Errorf("could not get profile templates: %w", err)
	}

	return profiles.NewServicer(tmpls, pool, querier), nil
}

func createAuthenticationServicer(authenticator authenticatorPkg.Authenticator, pool *pgxpool.Pool, querier storage.Querier) (auth.Servicer, error) {
	tmpls, err := templates.GetTemplates(templates.Authentication)
	if err != nil {
		return nil, fmt.Errorf("could not get authentication templates: %w", err)
	}

	return auth.NewServicer(authenticator, tmpls, pool, querier), nil
}

func createSubmissionsServicer(broker submissions.Broker, pool *pgxpool.Pool, querier storage.Querier) (submissions.Servicer, error) {
	tmpls, err := templates.GetTemplates(templates.Submissions)
	if err != nil {
		return nil, fmt.Errorf("could not get submissions templates: %w", err)
	}

	return submissions.NewServicer(broker, tmpls, querier, pool), nil
}
