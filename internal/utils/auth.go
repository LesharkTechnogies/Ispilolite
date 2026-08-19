package utils

import (
	"errors"

	"github.com/dgrijalva/jwt-go"
	"golang.org/x/crypto/bcrypt"
)

// HashPassword hashes a plaintext password using bcrypt.
func HashPassword(password string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(hash), nil
}

// VerifyPassword compares a plaintext password against a bcrypt hash.
func VerifyPassword(password, hash string) (bool, error) {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	if err != nil {
		if errors.Is(err, bcrypt.ErrMismatchedHashAndPassword) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

// TokenClaims is the JWT claim set used for access and refresh tokens.
type TokenClaims struct {
	UserID    string `json:"user_id"`
	Role      string `json:"role"`
	TokenUse  string `json:"token_use"`
	Issuer    string `json:"iss"`
	Subject   string `json:"sub"`
	Expires   int64  `json:"exp"`
	NotBefore int64  `json:"nbf"`
	IssuedAt  int64  `json:"iat"`
	ID        string `json:"jti"`
}

// jwtClaims adapts TokenClaims to jwt-go's expected Claims interface via
// embedded StandardClaims, keeping compatibility with the dgrijalva/jwt-go
// library already used in internal/middleware/auth.go.
type jwtClaims struct {
	UserID   string `json:"user_id"`
	Role     string `json:"role"`
	TokenUse string `json:"token_use"`
	jwt.StandardClaims
}

// SignToken signs a TokenClaims set into a compact JWT string.
func SignToken(secret []byte, claims TokenClaims) (string, error) {
	c := jwtClaims{
		UserID:   claims.UserID,
		Role:     claims.Role,
		TokenUse: claims.TokenUse,
		StandardClaims: jwt.StandardClaims{
			Id:        claims.ID,
			Issuer:    claims.Issuer,
			Subject:   claims.Subject,
			ExpiresAt: claims.Expires,
			NotBefore: claims.NotBefore,
			IssuedAt:  claims.IssuedAt,
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, c)
	return token.SignedString(secret)
}

// ParseToken parses and validates a JWT string, returning its TokenClaims.
func ParseToken(secret []byte, tokenString string) (*TokenClaims, error) {
	c := &jwtClaims{}
	token, err := jwt.ParseWithClaims(tokenString, c, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("unexpected signing method")
		}
		return secret, nil
	})
	if err != nil {
		return nil, err
	}
	if !token.Valid {
		return nil, errors.New("invalid token")
	}
	return &TokenClaims{
		UserID:    c.UserID,
		Role:      c.Role,
		TokenUse:  c.TokenUse,
		Issuer:    c.Issuer,
		Subject:   c.Subject,
		Expires:   c.ExpiresAt,
		NotBefore: c.NotBefore,
		IssuedAt:  c.IssuedAt,
		ID:        c.Id,
	}, nil
}