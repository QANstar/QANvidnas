package auth

import (
	"crypto/rand"
	"database/sql"
	"errors"
	"fmt"
	"math/big"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

var (
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrUserExists         = errors.New("user already exists")
	ErrSetupDone          = errors.New("setup already completed")
	ErrInvalidInviteCode  = errors.New("invalid invite code")
	ErrInviteCodeUsed     = errors.New("invite code exhausted")
	ErrInviteCodeExpired  = errors.New("invite code expired")
)

type JWTClaims struct {
	UserID  int64  `json:"user_id"`
	IsAdmin bool   `json:"is_admin"`
	jwt.RegisteredClaims
}

type Service struct {
	db          *sql.DB
	jwtSecret   []byte
	tokenExpiry time.Duration
}

func NewService(db *sql.DB, jwtSecret string) *Service {
	return &Service{
		db:          db,
		jwtSecret:   []byte(jwtSecret),
		tokenExpiry: 7 * 24 * time.Hour,
	}
}

func (s *Service) HasAdmin() (bool, error) {
	var count int
	err := s.db.QueryRow("SELECT COUNT(*) FROM users WHERE is_admin = 1").Scan(&count)
	return count > 0, err
}

func (s *Service) SetupAdmin(username, password string) (string, error) {
	hasAdmin, err := s.HasAdmin()
	if err != nil {
		return "", err
	}
	if hasAdmin {
		return "", ErrSetupDone
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", fmt.Errorf("hash password: %w", err)
	}

	res, err := s.db.Exec(
		"INSERT INTO users (username, password_hash, is_admin) VALUES (?, ?, 1)",
		username, string(hash),
	)
	if err != nil {
		return "", fmt.Errorf("create admin: %w", err)
	}

	userID, _ := res.LastInsertId()
	return s.generateToken(userID, true)
}

func (s *Service) Login(username, password string) (string, error) {
	var id int64
	var hash string
	var isAdmin bool

	err := s.db.QueryRow(
		"SELECT id, password_hash, is_admin FROM users WHERE username = ?",
		username,
	).Scan(&id, &hash, &isAdmin)

	if err == sql.ErrNoRows {
		return "", ErrInvalidCredentials
	}
	if err != nil {
		return "", fmt.Errorf("query user: %w", err)
	}

	if err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)); err != nil {
		return "", ErrInvalidCredentials
	}

	return s.generateToken(id, isAdmin)
}

func (s *Service) Register(inviteCode, username, password string) (string, error) {
	// Validate invite code
	var maxUses, used int
	var expiresAt string
	err := s.db.QueryRow(
		"SELECT max_uses, used, expires_at FROM invite_codes WHERE code = ?",
		inviteCode,
	).Scan(&maxUses, &used, &expiresAt)

	if err == sql.ErrNoRows {
		return "", ErrInvalidInviteCode
	}
	if err != nil {
		return "", fmt.Errorf("query invite code: %w", err)
	}

	if maxUses > 0 && used >= maxUses {
		return "", ErrInviteCodeUsed
	}

	if expiresAt != "" {
		expiry, err := time.Parse("2006-01-02", expiresAt)
		if err == nil && time.Now().After(expiry) {
			return "", ErrInviteCodeExpired
		}
	}

	// Check username uniqueness
	var existingID int64
	err = s.db.QueryRow("SELECT id FROM users WHERE username = ?", username).Scan(&existingID)
	if err == nil {
		return "", ErrUserExists
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", fmt.Errorf("hash password: %w", err)
	}

	tx, err := s.db.Begin()
	if err != nil {
		return "", fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	res, err := tx.Exec(
		"INSERT INTO users (username, password_hash, is_admin) VALUES (?, ?, 0)",
		username, string(hash),
	)
	if err != nil {
		return "", fmt.Errorf("create user: %w", err)
	}

	userID, _ := res.LastInsertId()

	// Increment invite code usage
	if _, err := tx.Exec("UPDATE invite_codes SET used = used + 1 WHERE code = ?", inviteCode); err != nil {
		return "", fmt.Errorf("update invite code: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return "", fmt.Errorf("commit: %w", err)
	}

	return s.generateToken(userID, false)
}

func (s *Service) ValidateToken(tokenStr string) (*JWTClaims, error) {
	token, err := jwt.ParseWithClaims(tokenStr, &JWTClaims{}, func(t *jwt.Token) (interface{}, error) {
		return s.jwtSecret, nil
	})
	if err != nil {
		return nil, err
	}

	claims, ok := token.Claims.(*JWTClaims)
	if !ok || !token.Valid {
		return nil, errors.New("invalid token")
	}

	return claims, nil
}

func (s *Service) generateToken(userID int64, isAdmin bool) (string, error) {
	claims := &JWTClaims{
		UserID:  userID,
		IsAdmin: isAdmin,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(s.tokenExpiry)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(s.jwtSecret)
}

// GenerateDeviceCode creates a 4-character device code for TV login.
func (s *Service) GenerateDeviceCode() (string, error) {
	const charset = "ABCDEFGHJKLMNPQRSTUVWXYZ23456789" // Removed ambiguous chars
	code := make([]byte, 4)
	for i := range code {
		n, err := rand.Int(rand.Reader, big.NewInt(int64(len(charset))))
		if err != nil {
			return "", err
		}
		code[i] = charset[n.Int64()]
	}

	expiresAt := time.Now().Add(10 * time.Minute)
	_, err := s.db.Exec(
		"INSERT INTO device_codes (code, expires_at) VALUES (?, ?)",
		string(code), expiresAt,
	)
	if err != nil {
		return "", fmt.Errorf("insert device code: %w", err)
	}

	return string(code), nil
}

// AuthorizeDeviceCode marks a device code as authorized for a user.
func (s *Service) AuthorizeDeviceCode(code string, userID int64) error {
	result, err := s.db.Exec(
		"UPDATE device_codes SET user_id = ?, used = 1 WHERE code = ? AND used = 0 AND expires_at > ?",
		userID, code, time.Now(),
	)
	if err != nil {
		return fmt.Errorf("authorize device code: %w", err)
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		return errors.New("invalid or expired device code")
	}

	return nil
}

// PollDeviceCode checks if a device code has been authorized and returns a token.
func (s *Service) PollDeviceCode(code string) (string, error) {
	var userID int64
	var used bool
	err := s.db.QueryRow(
		"SELECT user_id, used FROM device_codes WHERE code = ? AND expires_at > ?",
		code, time.Now(),
	).Scan(&userID, &used)

	if err == sql.ErrNoRows {
		return "", errors.New("device code expired")
	}
	if err != nil {
		return "", fmt.Errorf("query device code: %w", err)
	}

	if !used || userID == 0 {
		return "", nil // Not yet authorized
	}

	// Get user info
	var isAdmin bool
	err = s.db.QueryRow("SELECT is_admin FROM users WHERE id = ?", userID).Scan(&isAdmin)
	if err != nil {
		return "", fmt.Errorf("query user: %w", err)
	}

	return s.generateToken(userID, isAdmin)
}
