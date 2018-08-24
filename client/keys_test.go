package client

import (
	"crypto/rsa"
	"fmt"
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"

	"golang.org/x/crypto/ed25519"
)

func TestGenerateKeys(t *testing.T) {
	var tests = []struct {
		keytype string
		keysize int
		want    string
	}{
		{"", 0, "*rsa.PrivateKey"},
		{"rsa", 1024, "*rsa.PrivateKey"},
		{"rsa", 0, "*rsa.PrivateKey"},
		{"ecdsa", 0, "*ecdsa.PrivateKey"},
		{"ecdsa", 384, "*ecdsa.PrivateKey"},
		{"ed25519", 0, "*ed25519.PrivateKey"},
	}

	for _, tst := range tests {
		var k Key
		var err error
		k, _, err = GenerateKey(KeyType(tst.keytype), KeySize(tst.keysize))
		if err != nil {
			t.Error(err)
		}
		if reflect.TypeOf(k).String() != tst.want {
			t.Errorf("Wrong key type returned. Got %T, wanted %s", k, tst.want)
		}
	}
}

func TestDefaultOptions(t *testing.T) {
	k, _, err := GenerateKey()
	if err != nil {
		t.Error(err)
	}
	_, ok := k.(*rsa.PrivateKey)
	if !ok {
		t.Errorf("Unexpected key type %T, wanted *rsa.PrivateKey", k)
	}
}

func TestGenerateKeyType(t *testing.T) {
	k, _, err := GenerateKey(KeyType("ed25519"))
	if err != nil {
		t.Error(err)
	}
	_, ok := k.(*ed25519.PrivateKey)
	if !ok {
		t.Errorf("Unexpected key type %T, wanted *ed25519.PrivateKey", k)
	}
}

func TestGenerateKeySize(t *testing.T) {
	k, _, err := GenerateKey(KeySize(1024))
	if err != nil {
		t.Error(err)
	}
	_, ok := k.(*rsa.PrivateKey)
	if !ok {
		t.Errorf("Unexpected key type %T, wanted *rsa.PrivateKey", k)
	}
}

func TestPEMEncode(t *testing.T) {
	tests := []struct {
		key      string
		expected string
	}{
		{"rsa", "RSA PRIVATE KEY"},
		{"ed25519", "OPENSSH PRIVATE KEY"},
		{"ecdsa", "EC PRIVATE KEY"},
	}
	for _, tt := range tests {
		key, _, _ := GenerateKey(KeyType(tt.key))
		pem, _ := pemBlockForKey(key)
		if tt.expected != pem.Type {
			t.Errorf("key %s: want %s, got %s", key, tt.expected, pem.Type)
		}
	}
	_, err := pemBlockForKey("blabbedy")
	if err == nil {
		t.Errorf("Got nil, expected an error")
	}
}

func TestECDSASizes(t *testing.T) {
	tests := []struct {
		size     int
		expected error
	}{
		{256, nil},
		{384, nil},
		{521, nil},
		{999, fmt.Errorf("")},
	}
	for _, tt := range tests {
		_, err := generateECDSAKey(tt.size)
		assert.IsType(t, tt.expected, err)
	}
}
