package auth

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/crypto/bcrypt"

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
	authenticator Authenticator
	templates     *templates.Templates
	pool          *pgxpool.Pool
	querier       storage.Querier
}

func NewServicer(authenticator Authenticator,
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
		http.Error(w, "could not render", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}

func (s *DefaultServicer) Login(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Invalid form data", http.StatusBadRequest)
		return
	}

	username := r.PostFormValue("username")
	password := r.PostFormValue("password")

	if username == "" || password == "" {
		http.Error(w, "username and password are required", http.StatusBadRequest)
		return
	}

	if len(password) < 8 {
		http.Error(w, "password length can not be less than 8", http.StatusBadRequest)
		return
	}

	s.loginUser(w, r, username, password)
}

func (s *DefaultServicer) loginUser(w http.ResponseWriter, r *http.Request, username string, password string) {
	user, err := s.querier.GetUserByUsername(r.Context(), s.pool, username)
	if err != nil {
		http.Error(w, "Username not found", http.StatusUnauthorized)
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		http.Error(w, "Invalid credentials", http.StatusUnauthorized)
		return
	}

	claims := Claims{
		UserID: user.ID.String(),
	}

	tokenString, tokenClaims, err := s.authenticator.GenerateToken(r.Context(), claims)
	if err != nil {
		http.Error(w, "Error generating token", http.StatusInternalServerError)
		return
	}

	cookie := &http.Cookie{
		Name:     TokenCookieKey,
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
		http.Error(w, "All fields are required", http.StatusBadRequest)
		return
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		slog.ErrorContext(r.Context(), "could not process password", "error", err, "username", username, "password", password)
		http.Error(w, "Error processing password", http.StatusInternalServerError)
		return
	}

	_, err = s.querier.CreateUser(r.Context(), s.pool, username, string(hashedPassword))
	if err != nil {
		slog.ErrorContext(r.Context(), "could not create user", "error", err, "username", username)
		http.Error(w, "Could not create user", http.StatusInternalServerError)
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
