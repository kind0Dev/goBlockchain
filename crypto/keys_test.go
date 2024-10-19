package crypto

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGeneratePrivateKey(t *testing.T) {
	privKey := GeneratePrivateKey()
	assert.Equal(t, len(privKey.Bytes()), PrivKeyLen)

	pubKey := privKey.Public()
	assert.Equal(t, len(pubKey.Bytes()), PubKeyLen)
}

func TestNewPrivateKeyFromString(t *testing.T) {
	var (
		seed       = "1c8406a01c649c0dc9c679825c2b62424e78d3ee809e1d802fe5ef914aee1354"
		privKey    = NewPrivateKeyFromString(seed)
		addressStr = "1d17d7ce0b483771fc7629f41515b786220de36f"
	)

	assert.Equal(t, PrivKeyLen, len(privKey.Bytes()))
	address := privKey.Public().Address()
	assert.Equal(t, addressStr, address.String())

}

func TestPrivateKeySign(t *testing.T) {
	privKey := GeneratePrivateKey()
	pubKey := privKey.Public()
	msg := []byte("foo bar baz")

	sig := privKey.Sign(msg)
	assert.True(t, sig.Verify(pubKey, msg))

	// Test with invalid msg
	assert.False(t, sig.Verify(pubKey, []byte("foo")))

	// Test with invalid pubKey
	invalidPrivKey := GeneratePrivateKey()
	invalidPubKey := invalidPrivKey.Public()
	assert.False(t, sig.Verify(invalidPubKey, msg))
}

func TestPublicKeyToAddress(t *testing.T) {
	privKey := GeneratePrivateKey()
	pubKey := privKey.Public()
	address := pubKey.Address()
	assert.Equal(t, AddressLen, len(address.Bytes()))
	fmt.Println(address)
}
