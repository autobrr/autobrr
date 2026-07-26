// Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package argon2id

import (
	"regexp"
	"strings"
	"testing"
)

func TestCreateHash(t *testing.T) {
	t.Parallel()
	hashRX, err := regexp.Compile(`^\$argon2id\$v=19\$m=65536,t=1,p=2\$[A-Za-z0-9+/]{22}\$[A-Za-z0-9+/]{43}$`)
	if err != nil {
		t.Fatal(err)
	}

	hash1, err := CreateHash("pa$$word", DefaultParams)
	if err != nil {
		t.Fatal(err)
	}

	if !hashRX.MatchString(hash1) {
		t.Errorf("hash %q not in correct format", hash1)
	}

	hash2, err := CreateHash("pa$$word", DefaultParams)
	if err != nil {
		t.Fatal(err)
	}

	if strings.Compare(hash1, hash2) == 0 {
		t.Error("hashes must be unique")
	}
}

func TestComparePasswordAndHash(t *testing.T) {
	t.Parallel()
	hash, err := CreateHash("pa$$word", DefaultParams)
	if err != nil {
		t.Fatal(err)
	}

	match, err := ComparePasswordAndHash("pa$$word", hash)
	if err != nil {
		t.Fatal(err)
	}

	if !match {
		t.Error("expected password and hash to match")
	}

	match, err = ComparePasswordAndHash("otherPa$$word", hash)
	if err != nil {
		t.Fatal(err)
	}

	if match {
		t.Error("expected password and hash to not match")
	}
}

func TestDecodeHash(t *testing.T) {
	t.Parallel()
	hash, err := CreateHash("pa$$word", DefaultParams)
	if err != nil {
		t.Fatal(err)
	}

	params, _, _, err := DecodeHash(hash)
	if err != nil {
		t.Fatal(err)
	}
	if *params != *DefaultParams {
		t.Fatalf("expected %#v got %#v", *DefaultParams, *params)
	}
}

func TestCheckHash(t *testing.T) {
	t.Parallel()
	hash, err := CreateHash("pa$$word", DefaultParams)
	if err != nil {
		t.Fatal(err)
	}

	ok, params, err := CheckHash("pa$$word", hash)
	if err != nil {
		t.Fatal(err)
	}
	if !ok {
		t.Fatal("expected password to match")
	}
	if *params != *DefaultParams {
		t.Fatalf("expected %#v got %#v", *DefaultParams, *params)
	}
}

func TestStrictDecoding(t *testing.T) {
	t.Parallel()
	// "bug" valid hash: $argon2id$v=19$m=65536,t=1,p=2$UDk0zEuIzbt0x3bwkf8Bgw$ihSfHWUJpTgDvNWiojrgcN4E0pJdUVmqCEdRZesx9tE
	ok, _, err := CheckHash("bug", "$argon2id$v=19$m=65536,t=1,p=2$UDk0zEuIzbt0x3bwkf8Bgw$ihSfHWUJpTgDvNWiojrgcN4E0pJdUVmqCEdRZesx9tE")
	if err != nil {
		t.Fatal(err)
	}
	if !ok {
		t.Fatal("expected password to match")
	}

	// changed one last character of the hash
	ok, _, err = CheckHash("bug", "$argon2id$v=19$m=65536,t=1,p=2$UDk0zEuIzbt0x3bwkf8Bgw$ihSfHWUJpTgDvNWiojrgcN4E0pJdUVmqCEdRZesx9tF")
	if err == nil {
		t.Fatal("Hash validation should fail")
	}

	if ok {
		t.Fatal("Hash validation should fail")
	}
}
