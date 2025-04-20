package authenticator

import "errors"

const (
	issuer         = "gojudge"
	TokenCookieKey = "token"
)

var ErrSigningKeyNotFound = errors.New("signing key not found")
