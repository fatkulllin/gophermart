package password

import (
	"bytes"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"

	"golang.org/x/crypto/scrypt"
)

const (
	SaltByteSize    = 16
	HashKeySize     = 32
	HashIterations  = 32768
	HashBlockSize   = 8
	HashParallelism = 1
)

var errInvalidPasswordHash = errors.New("password hash does not have the correct format")

type Password struct{}

func NewPassword() *Password {
	return &Password{}
}

func (pass *Password) Hash(password string) (string, error) {
	// Generate salt
	salt := make([]byte, SaltByteSize)
	if _, err := rand.Read(salt); err != nil {
		return "", fmt.Errorf("generating password salt: %w", err)
	}

	// Generate password hash
	passwordHash, err := scrypt.Key([]byte(password), salt, HashIterations, HashBlockSize, HashParallelism, HashKeySize)
	if err != nil {
		return "", fmt.Errorf("generating password salt: %w", err)
	}

	// Concatenate algorithm settings and hash with $ (this is a common format for scrypt hashes)
	base64Password := base64.StdEncoding.EncodeToString(passwordHash)
	base64Salt := base64.StdEncoding.EncodeToString(salt)

	return fmt.Sprintf("scrypt$%d$%d$%d$%s$%s", HashIterations, HashBlockSize, HashParallelism, base64Salt, base64Password), nil
}

// Compare compares a given password to an existing scrypt password hash
func (pass *Password) Compare(hash string, password string) (bool, error) {
	var n, r, p int
	var alg, originalHash, salt string

	if _, err := fmt.Sscanf(strings.ReplaceAll(hash, "$", " "), "%s %d %d %d %s %s", &alg, &n, &r, &p, &salt, &originalHash); err != nil {
		return false, errInvalidPasswordHash
	}

	hashBytes, err := base64.StdEncoding.DecodeString(originalHash)
	if err != nil {
		return false, fmt.Errorf("decoding password hash: %w", err)
	}

	saltBytes, err := base64.StdEncoding.DecodeString(salt)
	if err != nil {
		return false, fmt.Errorf("decoding salt: %w", err)
	}

	passwordHash, err := scrypt.Key([]byte(password), saltBytes, n, r, p, len(hashBytes))
	if err != nil {
		return false, fmt.Errorf("generating password salt: %w", err)
	}

	return bytes.Equal(hashBytes, passwordHash), nil
}
