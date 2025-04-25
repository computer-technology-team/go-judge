package auth

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/crypto/bcrypt"

	"github.com/computer-technology-team/go-judge/internal/auth/authenticator"
	"github.com/computer-technology-team/go-judge/internal/storage"
	"github.com/computer-technology-team/go-judge/web/templates"
)

type Servicer interface {
	ShowLoginPage(w http.ResponseWriter, r *http.Request)
	Login(w http.ResponseWriter, r *http.Request)
	ShowSignupPage(w http.ResponseWriter, r *http.Request)
	Signup(w http.ResponseWriter, r *http.Request)
	Logout(w http.ResponseWriter, r *http.Request)
}

type DefaultServicer struct {
	authenticator authenticator.Authenticator
	templates     *templates.Templates
	pool          *pgxpool.Pool
	querier       storage.Querier
}

func NewServicer(authenticator authenticator.Authenticator,
	templates *templates.Templates,
	pool *pgxpool.Pool,
	querier storage.Querier,
) Servicer {
	return &DefaultServicer{
		templates:     templates,
		authenticator: authenticator,
		pool:          pool,
		querier:       querier,
	}
}

// ShowLoginPage handles user login
func (s *DefaultServicer) ShowLoginPage(w http.ResponseWriter, r *http.Request) {
	err := s.templates.Render(r.Context(), "login", w, nil)
	if err != nil {
		slog.Error("could not render login", "error", err)
		templates.RenderError(r.Context(), w, "could not render", http.StatusInternalServerError, s.templates)
		return
	}
	w.WriteHeader(http.StatusOK)
}

func (s *DefaultServicer) Login(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		templates.RenderError(r.Context(), w, "Invalid form data", http.StatusBadRequest, s.templates)
		return
	}

	username := r.PostFormValue("username")
	password := r.PostFormValue("password")

	if username == "" || password == "" {
		templates.RenderError(r.Context(), w, "username and password are required", http.StatusBadRequest, s.templates)
		return
	}

	if len(password) < 8 {
		templates.RenderError(r.Context(), w, "password length can not be less than 8", http.StatusBadRequest, s.templates)
		return
	}

	s.loginUser(w, r, username, password)
}

func (s *DefaultServicer) loginUser(w http.ResponseWriter, r *http.Request, username string, password string) {
	user, err := s.querier.GetUserByUsername(r.Context(), s.pool, username)
	if err != nil {
		templates.RenderError(r.Context(), w, "Username not found", http.StatusUnauthorized, s.templates)
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		templates.RenderError(r.Context(), w, "Invalid credentials", http.StatusUnauthorized, s.templates)
		return
	}

	claims := authenticator.Claims{
		UserID: user.ID.String(),
	}

	tokenString, tokenClaims, err := s.authenticator.GenerateToken(r.Context(), claims)
	if err != nil {
		templates.RenderError(r.Context(), w, "Error generating token", http.StatusInternalServerError, s.templates)
		return
	}

	cookie := &http.Cookie{
		Name:     authenticator.TokenCookieKey,
		Value:    tokenString,
		Expires:  tokenClaims.ExpiresAt.Time,
		HttpOnly: true,
		Path:     "/",
	}
	http.SetCookie(w, cookie)

	http.Redirect(w, r, "/", http.StatusSeeOther)

	w.WriteHeader(http.StatusOK)
}

// ShowSignupPage handles user registration
func (s *DefaultServicer) ShowSignupPage(w http.ResponseWriter, r *http.Request) {
	err := s.templates.Render(r.Context(), "signup", w, nil)
	if err != nil {
		slog.Error("could not render signup", "error", err)
		http.Error(w, "could not render", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func (s *DefaultServicer) Signup(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Invalid form data", http.StatusBadRequest)
		return
	}

	username := r.PostFormValue("username")
	password := r.PostFormValue("password")

	if username == "" || password == "" {
		templates.RenderError(r.Context(), w, "All fields are required", http.StatusBadRequest, s.templates)
		return
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		slog.ErrorContext(r.Context(), "could not process password", "error", err, "username", username, "password", password)
		templates.RenderError(r.Context(), w, "Error processing password", http.StatusInternalServerError, s.templates)
		return
	}

	_, err = s.querier.CreateUser(r.Context(), s.pool, username, string(hashedPassword))
	if err != nil {
		slog.ErrorContext(r.Context(), "could not create user", "error", err, "username", username)
		templates.RenderError(r.Context(), w, "Could not create user", http.StatusInternalServerError, s.templates)
		return
	}

	s.loginUser(w, r, username, password)
}

// Logout handles user logout
func (s *DefaultServicer) Logout(w http.ResponseWriter, r *http.Request) {
	http.SetCookie(w, &http.Cookie{
		Name:     "token",
		Value:    "", // Clear the value
		Path:     "/",
		Expires:  time.Unix(0, 0), // Expire it
		MaxAge:   -1,              // Delete immediately
		HttpOnly: true,
	})

	http.Redirect(w, r, "/", http.StatusSeeOther)

	w.WriteHeader(http.StatusOK)
}
