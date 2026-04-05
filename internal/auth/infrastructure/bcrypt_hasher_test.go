package infrastructure

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBcryptHasher_Hash_ReturnsHashedPassword(t *testing.T) {
	hasher := NewBcryptHasher()

	hash, err := hasher.Hash("password123")

	require.NoError(t, err)
	assert.NotEqual(t, "password123", hash)
	assert.True(t, strings.HasPrefix(hash, "$2a$"))
}

func TestBcryptHasher_Compare_ValidPassword(t *testing.T) {
	hasher := NewBcryptHasher()

	hash, err := hasher.Hash("password123")
	require.NoError(t, err)

	err = hasher.Compare(hash, "password123")
	assert.NoError(t, err)
}

func TestBcryptHasher_Compare_InvalidPassword(t *testing.T) {
	hasher := NewBcryptHasher()

	hash, err := hasher.Hash("password123")
	require.NoError(t, err)

	err = hasher.Compare(hash, "wrong-password")
	assert.Error(t, err)
}
