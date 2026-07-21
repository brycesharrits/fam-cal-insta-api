package auth

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"time"

	"github.com/brycesharrits/fam-cal-insta/internal/domain"
	"github.com/lestrrat-go/jwx/v3/jwt"
)

const (
	googleJWKSURL    = "https://www.googleapis.com/oauth2/v3/certs"
	googleJWKSMaxAge = time.Hour
)

// Google publishes tokens under both variants of the issuer string.
var googleIssuers = []string{
	"https://accounts.google.com",
	"accounts.google.com",
}

type GoogleSignInVerifier struct {
	audience string
	jwks     *jwksCache
}

func NewGoogleSignInVerifier(audience string) *GoogleSignInVerifier {
	return &GoogleSignInVerifier{
		audience: audience,
		jwks:     newJWKSCache(googleJWKSURL, googleJWKSMaxAge),
	}
}

func (v *GoogleSignInVerifier) Provider() string {
	return domain.ProviderGoogle
}

func (v *GoogleSignInVerifier) Verify(ctx context.Context, idToken string) (*OIDCClaims, error) {
	set, err := v.jwks.Get(ctx)
	if err != nil {
		return nil, fmt.Errorf("google JWKS: %w", err)
	}

	tok, err := jwt.ParseString(idToken,
		jwt.WithKeySet(set),
		jwt.WithAudience(v.audience),
		jwt.WithValidate(true),
	)
	if err != nil {
		return nil, fmt.Errorf("google token verify: %w", err)
	}

	iss, _ := tok.Issuer()
	if !slices.Contains(googleIssuers, iss) {
		return nil, fmt.Errorf("unexpected google issuer: %s", iss)
	}

	sub, _ := tok.Subject()
	if sub == "" {
		return nil, errors.New("google token missing sub claim")
	}

	claims := &OIDCClaims{Subject: sub}

	var email string
	if err := tok.Get("email", &email); err == nil {
		claims.Email = email
	}
	var emailVerified bool
	if err := tok.Get("email_verified", &emailVerified); err == nil {
		claims.EmailVerified = emailVerified
	}

	return claims, nil
}
