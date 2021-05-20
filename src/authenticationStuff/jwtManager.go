package authentication

import (
	"fmt"
	"time"

	"github.com/dgrijalva/jwt-go"
)

type JWTManager struct {
	/* This struct is a JSON web token (JWT) manager, it
	describes the info of the JWT */
	SecretKey     string
	TokenDuration time.Duration
}

type UserClaims struct {
	/* This is a custom JWT claim that describes the information
	that a JWT will contain about the user */
	jwt.StandardClaims
	Username string `json:"username"`
	Role     string `json:"role"`
}

func NewJWTManager(sercretKey string, tokenDuration time.Duration) *JWTManager {
	// This function returns a new JWT manager
	return &JWTManager{sercretKey, tokenDuration}
}

func (manager *JWTManager) GenerateManager(user *User) (string, error) {
	// This function generates and returns a signed JWT
	claims := UserClaims{
		StandardClaims: jwt.StandardClaims{
			ExpiresAt: time.Now().Add(manager.TokenDuration).Unix(),
		},
		Username: user.Username,
		Role:     user.Role,
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims) // Consider using something a bit stronger for production
	return token.SignedString([]byte(manager.SecretKey))
}

func (manager *JWTManager) VerifyJWT(accessToken string) (*UserClaims, error) {
	// This function verifies the provided JWT
	token, err := jwt.ParseWithClaims(
		accessToken,
		&UserClaims{},
		func(token *jwt.Token) (interface{}, error) {
			_, ok := token.Method.(*jwt.SigningMethodHMAC)
			if !ok {
				return nil, fmt.Errorf("unexpected token signing method")
			}
			return []byte(manager.SecretKey), nil
		},
	)

	if err != nil {
		return nil, fmt.Errorf("invalid token: %v", err)
	}

	claims, ok := token.Claims.(*UserClaims)
	if !ok {
		return nil, fmt.Errorf("invalid token claims")
	}

	return claims, nil
}
