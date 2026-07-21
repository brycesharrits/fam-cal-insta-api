package auth

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/brycesharrits/fam-cal-insta/internal/domain"
	"github.com/lestrrat-go/jwx/v3/jwt"
)

const (
	appleJWKSURL   = "https://appleid.apple.com/auth/keys"
	appleIssuer    = "https://appleid.apple.com"
	appleJWKSMaxAge = time.Hour
)

type AppleSignInVerifier struct {
	audience string
	jwks     *jwksCache
}

func NewAppleSignInVerifier(audience string) *AppleSignInVerifier {
	return &AppleSignInVerifier{
		audience: audience,
		jwks:     newJWKSCache(appleJWKSURL, appleJWKSMaxAge),
	}
}

func (v *AppleSignInVerifier) Provider() string {
	return domain.ProviderApple
}

func (v *AppleSignInVerifier) Verify(ctx context.Context, idToken string) (*OIDCClaims, error) {
	set, err := v.jwks.Get(ctx)
	if err != nil {
		return nil, fmt.Errorf("apple JWKS: %w", err)
	}

	tok, err := jwt.ParseString(idToken,
		jwt.WithKeySet(set),
		jwt.WithIssuer(appleIssuer),
		jwt.WithAudience(v.audience),
		jwt.WithValidate(true),
	)
	if err != nil {
		return nil, fmt.Errorf("apple token verify: %w", err)
	}

	sub, _ := tok.Subject()
	if sub == "" {
		return nil, errors.New("apple token missing sub claim")
	}

	claims := &OIDCClaims{Subject: sub}

	var email string
	if err := tok.Get("email", &email); err == nil {
		claims.Email = email
	}
	// Apple encodes email_verified as either bool or string — try both.
	var emailVerifiedBool bool
	if err := tok.Get("email_verified", &emailVerifiedBool); err == nil {
		claims.EmailVerified = emailVerifiedBool
	} else {
		var emailVerifiedStr string
		if err := tok.Get("email_verified", &emailVerifiedStr); err == nil {
			claims.EmailVerified = emailVerifiedStr == "true"
		}
	}

	return claims, nil
}
