package auth

import (
	"time"

	"github.com/golang-jwt/jwt/v5"

	"github.com/sakid00/massmaker-be/internal/domain"
)

const (
	TypAccess  = "access"
	TypService = "service"

	AudEnmasse   = "enmasse"
	AudMassmaker = "massmaker"
	IssMassmaker = "massmaker"
	SubMassmaker = "massmaker-bff"
)

type AccessClaims struct {
	jwt.RegisteredClaims
	Role string `json:"role"`
	Typ  string `json:"typ"`
}

type ServiceClaims struct {
	jwt.RegisteredClaims
	Typ string `json:"typ"`
}

func ParseAccess(secret, issuer, token string) (*AccessClaims, error) {
	parsed, err := jwt.ParseWithClaims(token, &AccessClaims{}, func(t *jwt.Token) (any, error) {
		if t.Method != jwt.SigningMethodHS256 {
			return nil, domain.ErrUnauthorized
		}
		return []byte(secret), nil
	}, jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}), jwt.WithIssuer(issuer), jwt.WithAudience(AudMassmaker))
	if err != nil || !parsed.Valid {
		return nil, domain.ErrUnauthorized
	}
	claims, ok := parsed.Claims.(*AccessClaims)
	if !ok || claims.Typ != TypAccess {
		return nil, domain.ErrUnauthorized
	}
	return claims, nil
}

func MintService(secret string, ttl time.Duration) (string, error) {
	now := time.Now()
	tok := jwt.NewWithClaims(jwt.SigningMethodHS256, ServiceClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    IssMassmaker,
			Subject:   SubMassmaker,
			Audience:  jwt.ClaimStrings{AudEnmasse},
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(ttl)),
		},
		Typ: TypService,
	})
	return tok.SignedString([]byte(secret))
}
