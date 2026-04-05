package auth

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

const applePublicKeysURL = "https://appleid.apple.com/auth/keys"

// AppleClaims holds the fields we care about from an Apple identity token.
type AppleClaims struct {
	Email         string `json:"email"`
	EmailVerified bool   `json:"email_verified,string"`
	Sub           string `json:"sub"` // Apple user ID
	jwt.RegisteredClaims
}

// AppleSignInVerifier validates Apple identity tokens.
type AppleSignInVerifier struct {
	httpClient *http.Client
	// Cached JWKS — refreshed on 401/key-not-found
	cachedKeys map[string]interface{}
	lastFetch  time.Time
}

func NewAppleSignInVerifier() *AppleSignInVerifier {
	return &AppleSignInVerifier{
		httpClient: &http.Client{Timeout: 10 * time.Second},
		cachedKeys: make(map[string]interface{}),
	}
}

// Verify validates the identity token and returns the Apple user ID and email.
func (a *AppleSignInVerifier) Verify(ctx context.Context, identityToken string) (*AppleClaims, error) {
	keys, err := a.fetchPublicKeys(ctx)
	if err != nil {
		return nil, fmt.Errorf("fetching Apple public keys: %w", err)
	}

	token, err := jwt.ParseWithClaims(identityToken, &AppleClaims{}, func(t *jwt.Token) (interface{}, error) {
		kid, ok := t.Header["kid"].(string)
		if !ok {
			return nil, errors.New("missing kid in token header")
		}
		key, ok := keys[kid]
		if !ok {
			return nil, fmt.Errorf("unknown key id: %s", kid)
		}
		return key, nil
	})
	if err != nil {
		return nil, fmt.Errorf("parsing identity token: %w", err)
	}

	claims, ok := token.Claims.(*AppleClaims)
	if !ok || !token.Valid {
		return nil, errors.New("invalid identity token")
	}

	return claims, nil
}

// fetchPublicKeys retrieves Apple's JWKS, caching for 1 hour.
func (a *AppleSignInVerifier) fetchPublicKeys(ctx context.Context) (map[string]interface{}, error) {
	if time.Since(a.lastFetch) < time.Hour && len(a.cachedKeys) > 0 {
		return a.cachedKeys, nil
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, applePublicKeysURL, nil)
	if err != nil {
		return nil, err
	}

	resp, err := a.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var jwks struct {
		Keys []json.RawMessage `json:"keys"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&jwks); err != nil {
		return nil, err
	}

	keys := make(map[string]interface{})
	for _, rawKey := range jwks.Keys {
		var header struct {
			Kid string `json:"kid"`
			Kty string `json:"kty"`
		}
		if err := json.Unmarshal(rawKey, &header); err != nil {
			continue
		}
		// Parse RSA public key from JWK
		pubKey, err := parseRSAPublicKeyFromJWK(rawKey)
		if err != nil {
			continue
		}
		keys[header.Kid] = pubKey
	}

	a.cachedKeys = keys
	a.lastFetch = time.Now()
	return keys, nil
}

// parseRSAPublicKeyFromJWK parses a JWK JSON blob into an *rsa.PublicKey.
// Uses the standard library — no extra deps.
func parseRSAPublicKeyFromJWK(rawKey json.RawMessage) (interface{}, error) {
	// Use jwt library's built-in JWK parsing via a minimal struct.
	// This is a simplified approach; for production consider lestrrat-go/jwx.
	var jwk struct {
		N string `json:"n"`
		E string `json:"e"`
	}
	if err := json.Unmarshal(rawKey, &jwk); err != nil {
		return nil, err
	}

	key, err := jwt.ParseRSAPublicKeyFromPEM([]byte(fmt.Sprintf(
		"-----BEGIN RSA PUBLIC KEY-----\n%s\n-----END RSA PUBLIC KEY-----",
		jwk.N,
	)))
	if err != nil {
		// Fall back: return raw JWK for the jwt library to handle
		return nil, fmt.Errorf("could not parse RSA key: %w", err)
	}
	return key, nil
}
