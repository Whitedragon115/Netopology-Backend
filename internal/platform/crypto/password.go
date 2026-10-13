// Package crypto holds the few primitives the app needs. Do not add home-made crypto here:
// only thin wrappers over x/crypto and the standard library.
package crypto

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"

	"golang.org/x/crypto/argon2"
)

// argon2id parameters (OWASP 2024 baseline: m=19 MiB, t=2, p=1). Stored in the hash string, so they
// can be raised later; NeedsRehash tells login to upgrade old hashes.
const (
	argonMemory  = 19 * 1024
	argonTime    = 2
	argonThreads = 1
	argonKeyLen  = 32
	argonSaltLen = 16
)

// Password length bounds (contract schema Password). Huma validates them from struct tags;
// these constants are for services that set passwords from other inputs.
const (
	MinPasswordLength = 12
	MaxPasswordLength = 256
)

// HashPassword returns a PHC-format string: $argon2id$v=19$m=19456,t=2,p=1$<salt>$<hash>.
func HashPassword(password string) (string, error) {
	salt := make([]byte, argonSaltLen)
	if _, err := rand.Read(salt); err != nil {
		return "", err
	}
	key := argon2.IDKey([]byte(password), salt, argonTime, argonMemory, argonThreads, argonKeyLen)
	b64 := base64.RawStdEncoding
	return fmt.Sprintf("$argon2id$v=%d$m=%d,t=%d,p=%d$%s$%s",
		argon2.Version, argonMemory, argonTime, argonThreads, b64.EncodeToString(salt), b64.EncodeToString(key)), nil
}

// VerifyPassword checks password against a hash from HashPassword in constant time.
func VerifyPassword(password, encoded string) (bool, error) {
	p, salt, key, err := decode(encoded)
	if err != nil {
		return false, err
	}
	got := argon2.IDKey([]byte(password), salt, p.time, p.memory, p.threads, uint32(len(key)))
	return subtle.ConstantTimeCompare(got, key) == 1, nil
}

// NeedsRehash reports whether the hash uses weaker parameters than the current ones.
func NeedsRehash(encoded string) bool {
	p, _, _, err := decode(encoded)
	return err != nil || p.memory < argonMemory || p.time < argonTime
}

// DummyVerify burns the same time as a real check. Call it when the username does not exist,
// so login timing does not reveal which usernames exist.
func DummyVerify(password string) {
	_ = argon2.IDKey([]byte(password), make([]byte, argonSaltLen), argonTime, argonMemory, argonThreads, argonKeyLen)
}

type params struct {
	memory  uint32
	time    uint32
	threads uint8
}

func decode(encoded string) (params, []byte, []byte, error) {
	var p params
	parts := strings.Split(encoded, "$")
	if len(parts) != 6 || parts[1] != "argon2id" {
		return p, nil, nil, errors.New("not an argon2id hash")
	}
	var version int
	if _, err := fmt.Sscanf(parts[2], "v=%d", &version); err != nil || version != argon2.Version {
		return p, nil, nil, errors.New("unsupported argon2 version")
	}
	if _, err := fmt.Sscanf(parts[3], "m=%d,t=%d,p=%d", &p.memory, &p.time, &p.threads); err != nil {
		return p, nil, nil, errors.New("bad argon2 parameters")
	}
	b64 := base64.RawStdEncoding
	salt, err := b64.DecodeString(parts[4])
	if err != nil {
		return p, nil, nil, err
	}
	key, err := b64.DecodeString(parts[5])
	if err != nil {
		return p, nil, nil, err
	}
	return p, salt, key, nil
}
