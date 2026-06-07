package auth

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"
)

const tokenTTL  = 7 * 24 * time.Hour
const bcryptCost = 12

type Claims struct {
	Sub      string `json:"sub"`
	Username string `json:"username"`
	IsAdmin  bool   `json:"is_admin"`
	Exp      int64  `json:"exp"`
	Iat      int64  `json:"iat"`
}

func GenerateToken(secret, userID, username string, isAdmin bool) (string, error) {
	header := base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"HS256","typ":"JWT"}`))
	payload, err := json.Marshal(Claims{
		Sub: userID, Username: username, IsAdmin: isAdmin,
		Exp: time.Now().Add(tokenTTL).Unix(),
		Iat: time.Now().Unix(),
	})
	if err != nil {
		return "", err
	}
	p := base64.RawURLEncoding.EncodeToString(payload)
	input := header + "." + p
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(input))
	return input + "." + base64.RawURLEncoding.EncodeToString(mac.Sum(nil)), nil
}

func VerifyToken(secret, token string) (*Claims, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return nil, fmt.Errorf("malformed token")
	}
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(parts[0] + "." + parts[1]))
	if !hmac.Equal(
		[]byte(base64.RawURLEncoding.EncodeToString(mac.Sum(nil))),
		[]byte(parts[2]),
	) {
		return nil, fmt.Errorf("invalid signature")
	}
	b, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return nil, err
	}
	var c Claims
	if err := json.Unmarshal(b, &c); err != nil {
		return nil, err
	}
	if time.Now().Unix() > c.Exp {
		return nil, fmt.Errorf("token expired")
	}
	return &c, nil
}

func HashPassword(password string) (string, error) {
	if len(password) < 8 {
		return "", fmt.Errorf("password must be at least 8 characters")
	}
	b, err := bcrypt.GenerateFromPassword([]byte(password), bcryptCost)
	return string(b), err
}

func CheckPassword(hash, password string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)) == nil
}
