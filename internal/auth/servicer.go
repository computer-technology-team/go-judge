package auth

import "net/http"

type Servicer interface {
	Login(w http.ResponseWriter, r *http.Request)
	Register(w http.ResponseWriter, r *http.Request)
	Logout(w http.ResponseWriter, r *http.Request)
	RefreshToken(w http.ResponseWriter, r *http.Request)
}

type DefaultServicer struct{}

func NewServicer() Servicer {
	return &DefaultServicer{}
}
