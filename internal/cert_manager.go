package internal

// Utilities for managing user certificates

import (
	"fmt"
	"strings"
	"time"

	"database/sql"
	"errors"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

// Gets the certificate of a given user. If one does not exist in the certificate db,mgenerate a new one by creating and hashing a uuid and insert it into the db.
func GetCertificate(username string, role string) string {
	var certificate string
	result := userDB.QueryRow("select certificate from users where username = ?", username)
	scanErr := result.Scan(&certificate)
	if scanErr != nil && !strings.Contains(scanErr.Error(), "NULL") {
		LogError(scanErr, "Problem scanning response to sql query SELECT certificate FROM users WHERE username = ? with args: "+username)
	}

	_, authenticated := VerifyCertificate(certificate)
	if certificate == "" || !authenticated {
		// Update user db
		newCertRaw := uuid.New()                                                       // Using uuid.new because it's a good source of randomness
		newCert, genErr := bcrypt.GenerateFromPassword([]byte(newCertRaw.String()), 6) // Pass through bcrypt to get a better format in my opinion

		if genErr != nil {
			LogError(genErr, "Problem generating new certificate from uuid "+newCertRaw.String())
		}

		_, err := userDB.Exec("update users set certificate = ? where username = ?", string(newCert), username)

		if err != nil {
			LogErrorf(err, "Problem executing sql query UPDATE users SET certificate = ? WHERE username = ? with args: %v, %v", newCert, username)
		}

		certificate = string(newCert)

		// Update certificate db
		_, execErr := authDB.Exec("insert into certs values(?,?,?)", string(newCert), role, username)
		if execErr != nil {
			LogErrorf(err, "Problem executing sql query INSERT INTO certs VALUES (?,?,?) with args: %v, %v, %v", newCert, role, username)
		}

	}

	return certificate
}

// Verifies the existence of a certificate
func VerifyCertificate(certificate string) (string, bool) {
	var certificateRole string
	result := authDB.QueryRow("select role from certs where certificate = ?", certificate)
	scanErr := result.Scan(&certificateRole)

	if scanErr != nil {
		// we can ignore errors from missing rows since thats cool here
		if !errors.Is(scanErr, sql.ErrNoRows) {
			LogError(scanErr, "error verifying certificate")
		}
		return "none", false
	}
	return certificateRole, true
}

type accessTokenClaims struct {
	UUID     string `json:"uuid"`
	Username string `json:"username"`
	Role     string `json:"role"`
	jwt.RegisteredClaims
}

func jwtSecret() ([]byte, error) {
	sec := "youmakeperkieimakepopyoutheactorjustlikerock" //strings.TrimSpace(os.Getenv("JWT_SECRET"))
	if sec == "" {
		return nil, errors.New("JWT_SECRET is not set")
	}
	if len(sec) < 32 {
		return nil, errors.New("JWT_SECRET too short (use 32+ chars)")
	}
	return []byte(sec), nil
}

func mintAccessToken(uuid, username, role string, ttl time.Duration) (string, error) {
	secret, err := jwtSecret()
	if err != nil {
		return "", err
	}

	now := time.Now()
	claims := accessTokenClaims{
		UUID:     uuid,
		Username: username,
		Role:     role,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   uuid,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(ttl)),
		},
	}

	tok := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return tok.SignedString(secret)
}

func parseBearerToken(authz string) (string, bool) {
	authz = strings.TrimSpace(authz)
	if authz == "" {
		return "", false
	}
	const pfx = "Bearer "
	if !strings.HasPrefix(authz, pfx) {
		return "", false
	}
	return strings.TrimSpace(authz[len(pfx):]), true
}

func verifyAccessToken(tokenString string) (*accessTokenClaims, error) {
	secret, err := jwtSecret()
	if err != nil {
		return nil, err
	}

	var claims accessTokenClaims
	tok, err := jwt.ParseWithClaims(tokenString, &claims, func(t *jwt.Token) (any, error) {
		if t.Method != jwt.SigningMethodHS256 {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return secret, nil
	})
	if err != nil {
		return nil, err
	}
	if !tok.Valid {
		return nil, errors.New("invalid token")
	}
	return &claims, nil
}
