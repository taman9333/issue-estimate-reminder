package github

import (
	"fmt"
	"os"
	"time"

	jwt "github.com/golang-jwt/jwt/v5"
	"github.com/taman9333/issue-estimate-reminder/internal/config"
)

type Auth struct {
	config *config.Config
}

func NewAuth(cfg *config.Config) *Auth {
	return &Auth{config: cfg}
}

func (a *Auth) GenerateJWT() (string, error) {
	keyData, err := os.ReadFile(a.config.PrivateKeyPath)
	if err != nil {
		return "", fmt.Errorf("failed to read private key: %v", err)
	}

	key, err := jwt.ParseRSAPrivateKeyFromPEM(keyData)
	if err != nil {
		return "", fmt.Errorf("failed to parse private key: %v", err)
	}

	now := time.Now()
	claims := jwt.MapClaims{
		"iss": a.config.AppID,
		"iat": now.Unix(),
		"exp": now.Add(10 * time.Minute).Unix(), // TODO: check later when this expires
	}

	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	tokenString, err := token.SignedString(key)
	if err != nil {
		return "", fmt.Errorf("failed to sign token: %v", err)
	}

	return tokenString, nil
}
