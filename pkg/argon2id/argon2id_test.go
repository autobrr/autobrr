// Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package argon2id

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCreateHash(t *testing.T) {
	t.Parallel()
	hashRX := `^\$argon2id\$v=19\$m=65536,t=1,p=2\$[A-Za-z0-9+/]{22}\$[A-Za-z0-9+/]{43}$`

	hash1, err := CreateHash("pa$$word", DefaultParams)
	require.NoError(t, err)

	assert.Regexp(t, hashRX, hash1)

	hash2, err := CreateHash("pa$$word", DefaultParams)
	require.NoError(t, err)

	assert.NotEqual(t, hash1, hash2, "hashes must be unique")
}

func TestComparePasswordAndHash(t *testing.T) {
	t.Parallel()
	hash, err := CreateHash("pa$$word", DefaultParams)
	require.NoError(t, err)

	match, err := ComparePasswordAndHash("pa$$word", hash)
	require.NoError(t, err)

	assert.True(t, match, "expected password and hash to match")

	match, err = ComparePasswordAndHash("otherPa$$word", hash)
	require.NoError(t, err)

	assert.False(t, match, "expected password and hash to not match")
}

func TestDecodeHash(t *testing.T) {
	t.Parallel()
	hash, err := CreateHash("pa$$word", DefaultParams)
	require.NoError(t, err)

	params, _, _, err := DecodeHash(hash)
	require.NoError(t, err)
	require.Equal(t, *DefaultParams, *params)
}

func TestCheckHash(t *testing.T) {
	t.Parallel()
	hash, err := CreateHash("pa$$word", DefaultParams)
	require.NoError(t, err)

	ok, params, err := CheckHash("pa$$word", hash)
	require.NoError(t, err)
	require.True(t, ok, "expected password to match")
	require.Equal(t, *DefaultParams, *params)
}

func TestStrictDecoding(t *testing.T) {
	t.Parallel()
	// "bug" valid hash: $argon2id$v=19$m=65536,t=1,p=2$UDk0zEuIzbt0x3bwkf8Bgw$ihSfHWUJpTgDvNWiojrgcN4E0pJdUVmqCEdRZesx9tE
	ok, _, err := CheckHash("bug", "$argon2id$v=19$m=65536,t=1,p=2$UDk0zEuIzbt0x3bwkf8Bgw$ihSfHWUJpTgDvNWiojrgcN4E0pJdUVmqCEdRZesx9tE")
	require.NoError(t, err)
	require.True(t, ok, "expected password to match")

	// changed one last character of the hash
	ok, _, err = CheckHash("bug", "$argon2id$v=19$m=65536,t=1,p=2$UDk0zEuIzbt0x3bwkf8Bgw$ihSfHWUJpTgDvNWiojrgcN4E0pJdUVmqCEdRZesx9tF")
	require.Error(t, err, "Hash validation should fail")

	require.False(t, ok, "Hash validation should fail")
}
