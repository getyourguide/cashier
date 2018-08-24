package signer

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"reflect"
	"testing"
	"time"

	"github.com/nsheridan/cashier/lib"
	"github.com/nsheridan/cashier/server/config"
	"github.com/nsheridan/cashier/server/store"
	"github.com/nsheridan/cashier/testdata"
	"github.com/stripe/krl"

	"golang.org/x/crypto/ssh"
)

var (
	signer *KeySigner
)

func TestMain(m *testing.M) {
	f, _ := ioutil.TempFile("", "ca_key")
	defer os.Remove(f.Name())
	ioutil.WriteFile(f.Name(), testdata.Priv, 0600)
	conf := &config.SSH{
		SigningKey:           f.Name(),
		MaxAge:               "1h",
		AdditionalPrincipals: []string{"ec2-user"},
		Permissions:          []string{"permit-pty", "force-command=/bin/ls"},
	}
	var err error
	signer, err = New(conf)
	if err != nil {
		fmt.Printf("Unable to create signer: %v\n", err)
		os.Exit(1)
	}
	os.Exit(m.Run())
}

func TestCert(t *testing.T) {
	r := &lib.SignRequest{
		Key:        string(testdata.Pub),
		ValidUntil: time.Now().Add(1 * time.Hour),
		Message:    "hello world",
	}
	cert, err := signer.SignUserKey(r, "gopher1")
	if err != nil {
		t.Error(err)
	}
	if !bytes.Equal(cert.SignatureKey.Marshal(), signer.ca.PublicKey().Marshal()) {
		t.Error("Cert signer and server signer don't match")
	}
	var principals []string
	principals = append(principals, "gopher1")
	principals = append(principals, signer.principals...)
	if !reflect.DeepEqual(cert.ValidPrincipals, principals) {
		t.Errorf("Expected %s, got %s", cert.ValidPrincipals, principals)
	}
	k1, _, _, _, err := ssh.ParseAuthorizedKey([]byte(r.Key))
	if err != nil {
		t.Errorf("Unable to parse key: %v", err)
	}
	k2 := cert.Key
	if !bytes.Equal(k1.Marshal(), k2.Marshal()) {
		t.Error("Cert key doesn't match public key")
	}
	if cert.ValidBefore != uint64(r.ValidUntil.Unix()) {
		t.Errorf("Invalid validity, expected %d, got %d", r.ValidUntil.Unix(), cert.ValidBefore)
	}
}

func TestMaxLifetime(t *testing.T) {
	// Request a cert valid for 1 week. Signer has a 1 hour limit.
	lifetime := time.Now().Add(7 * 24 * time.Hour)
	r := &lib.SignRequest{
		Key:        string(testdata.Pub),
		ValidUntil: lifetime,
	}
	cert, err := signer.SignUserKey(r, "gopher1")
	if err != nil {
		t.Error(err)
	}
	if uint64(lifetime.Unix()) == cert.ValidBefore {
		expires := time.Unix(int64(cert.ValidBefore), 0)
		expected := time.Now().Add(signer.maxLifetime)
		t.Errorf("Unexpected expiration: %v. Wanted %v", expires, expected)
	}
}

func TestRevocationList(t *testing.T) {
	r := &lib.SignRequest{
		Key:        string(testdata.Pub),
		ValidUntil: time.Now().Add(1 * time.Hour),
	}
	cert1, _ := signer.SignUserKey(r, "revoked")
	cert2, _ := signer.SignUserKey(r, "ok")
	var rec []*store.CertRecord
	rec = append(rec, &store.CertRecord{
		KeyID: cert1.KeyId,
	})
	rl, err := signer.GenerateRevocationList(rec)
	if err != nil {
		t.Error(err)
	}
	k, err := krl.ParseKRL(rl)
	if err != nil {
		t.Error(err)
	}
	if !k.IsRevoked(cert1) {
		t.Errorf("expected cert %s to be revoked", cert1.KeyId)
	}
	if k.IsRevoked(cert2) {
		t.Errorf("cert %s should not be revoked", cert2.KeyId)
	}
}

func TestPermissions(t *testing.T) {
	r := &lib.SignRequest{
		Key:        string(testdata.Pub),
		ValidUntil: time.Now().Add(1 * time.Hour),
	}
	cert, err := signer.SignUserKey(r, "gopher1")
	if err != nil {
		t.Error(err)
	}
	want := struct {
		extensions map[string]string
		options    map[string]string
	}{
		extensions: map[string]string{"permit-pty": ""},
		options:    map[string]string{"force-command": "/bin/ls"},
	}
	if !reflect.DeepEqual(cert.Extensions, want.extensions) {
		t.Errorf("Wrong permissions: wanted: %v got :%v", cert.Extensions, want.extensions)
	}
	if !reflect.DeepEqual(cert.CriticalOptions, want.options) {
		t.Errorf("Wrong options: wanted: %v got :%v", cert.CriticalOptions, want.options)
	}
}

func TestDefaultPermissions(t *testing.T) {
	r := &lib.SignRequest{
		Key:        string(testdata.Pub),
		ValidUntil: time.Now().Add(1 * time.Hour),
	}
	key, _ := ssh.ParsePrivateKey(testdata.Priv)
	signer := &KeySigner{
		ca:          key,
		maxLifetime: 12 * time.Hour,
	}
	cert, err := signer.SignUserKey(r, "gopher1")
	if err != nil {
		t.Error(err)
	}
	want := struct {
		extensions map[string]string
		options    map[string]string
	}{
		extensions: defaultPermissions,
		options:    map[string]string{},
	}
	if !reflect.DeepEqual(cert.Extensions, want.extensions) {
		t.Errorf("Wrong permissions: wanted: %v got :%v", cert.Extensions, want.extensions)
	}
	if !reflect.DeepEqual(cert.CriticalOptions, want.options) {
		t.Errorf("Wrong options: wanted: %v got :%v", cert.CriticalOptions, want.options)
	}
}
