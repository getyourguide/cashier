package client

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestConfig(t *testing.T) {
	a := assert.New(t)
	c, err := ReadConfig("testdata/test.conf")
	a.NoError(err)
	a.Equal(c.CA, "https://sshca.example.com")
	a.Equal(c.Keysize, 2048)
	a.Equal(c.Keytype, "rsa")
	a.Equal(c.Validity, "24h")
}
