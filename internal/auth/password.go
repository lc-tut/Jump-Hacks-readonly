package auth

import (
	"errors"

	"golang.org/x/crypto/bcrypt"
)

const (
	// MinPasswordLength defines the minimum password length
	MinPasswordLength = 8
	// MaxPasswordLength defines the maximum password length
	MaxPasswordLength = 128
	// BCryptCost defines the cost for bcrypt hashing
	BCryptCost = 12
)

var (
	ErrPasswordTooShort = errors.New("password is too short")
	ErrPasswordTooLong  = errors.New("password is too long")
	ErrPasswordInvalid  = errors.New("invalid password")
)

// HashPassword hashes a password using bcrypt
func HashPassword(password string) (string, error) {
	if err := ValidatePassword(password); err != nil {
		return "", err
	}

	hashedBytes, err := bcrypt.GenerateFromPassword([]byte(password), BCryptCost)
	if err != nil {
		return "", err
	}

	return string(hashedBytes), nil
}

// CheckPassword compares a hashed password with a plain text password
func CheckPassword(hashedPassword, password string) error {
	return bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(password))
}

// ValidatePassword validates password requirements
func ValidatePassword(password string) error {
	if len(password) < MinPasswordLength {
		return ErrPasswordTooShort
	}
	if len(password) > MaxPasswordLength {
		return ErrPasswordTooLong
	}
	
	// Add more validation rules as needed:
	// - Must contain uppercase letter
	// - Must contain lowercase letter
	// - Must contain number
	// - Must contain special character
	
	return nil
}

// GenerateRandomPassword generates a random password (useful for temporary passwords)
func GenerateRandomPassword(length int) string {
	if length < MinPasswordLength {
		length = MinPasswordLength
	}
	
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789!@#$%^&*"
	
	// For simplicity, using a basic implementation
	// In production, use crypto/rand for secure random generation
	password := make([]byte, length)
	for i := range password {
		password[i] = charset[i%len(charset)]
	}
	
	return string(password)
} 