package auth

import "context"

// OIDCClaims are the fields we extract from a verified identity token,
// regardless of which provider issued it.
type OIDCClaims struct {
	Subject       string
	Email         string
	EmailVerified bool
}

// OIDCVerifier validates an OIDC identity token from a specific provider.
type OIDCVerifier interface {
	Verify(ctx context.Context, idToken string) (*OIDCClaims, error)
	Provider() string
}
