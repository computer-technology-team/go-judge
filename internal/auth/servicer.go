package auth

import (
	"github.com/computer-technology-team/go-judge/internal/storage"
	"github.com/computer-technology-team/go-judge/web/templates"
	"github.com/jackc/pgx/v5/pgxpool"
	"net/http"
)

type Servicer interface {
	ShowLoginPage(w http.ResponseWriter, r *http.Request)
	Login(w http.ResponseWriter, r *http.Request)
	ShowSignupPage(w http.ResponseWriter, r *http.Request)
	Signup(w http.ResponseWriter, r *http.Request)

	Logout(w http.ResponseWriter, r *http.Request)
	RefreshToken(w http.ResponseWriter, r *http.Request)
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
	querier storage.Querier) Servicer {

	return &DefaultServicer{
		templates:     templates,
		authenticator: authenticator,
		pool:          pool,
		querier:       querier,
	}
}
