package auth

import (
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

const issuer = "asutport"

type Claims struct {
	Email     string `json:"email"`
	OrgID     string `json:"org_id"`
	Role      string `json:"role"`
	SessionID string `json:"session_id"`
	jwt.RegisteredClaims
}

func SignJWT(secret []byte, userID, orgID, sessionID uuid.UUID, email, role string, ttl time.Duration) (string, error) {
	if len(secret) < 16 {
		return "", errors.New("jwt secret too short")
	}
	now := time.Now()
	claims := Claims{
		Email:     email,
		OrgID:     orgID.String(),
		Role:      role,
		SessionID: sessionID.String(),
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    issuer,
			Subject:   userID.String(),
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(ttl)),
		},
	}
	t := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return t.SignedString(secret)
}

func ParseJWT(secret []byte, tokenString string) (*Claims, error) {
	t, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(t *jwt.Token) (any, error) {
		if t.Method != jwt.SigningMethodHS256 {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return secret, nil
	})
	if err != nil {
		return nil, err
	}
	claims, ok := t.Claims.(*Claims)
	if !ok || !t.Valid {
		return nil, errors.New("invalid token claims")
	}
	if claims.Issuer != issuer {
		return nil, errors.New("invalid issuer")
	}
	return claims, nil
}
