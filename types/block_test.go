package types

import (
	"encoding/hex"
	"fmt"
	"testing"

	"github.com/djeday123/blockchain2/crypto"
	"github.com/djeday123/blockchain2/util"
	"github.com/stretchr/testify/assert"
)

func TestSignVerifyBlock(t *testing.T) {
	var (
		block   = util.RandomBlock()
		privKey = crypto.GeneratePrivateKey()
		pubKey  = privKey.Public()
	)

	sig := SignBlock(privKey, block)
	assert.Equal(t, 64, len(sig.Bytes()))
	assert.True(t, sig.Verify(pubKey, HashBlock(block)))

	assert.Equal(t, block.PublicKey, pubKey.Bytes())
	assert.Equal(t, block.Signature, sig.Bytes())
	assert.True(t, VerifyBlock(block))

	invalidPrivKey := crypto.GeneratePrivateKey()
	block.PublicKey = invalidPrivKey.Public().Bytes()
	assert.False(t, VerifyBlock(block))
}

func TestHashBlock(t *testing.T) {
	block := util.RandomBlock()
	hash := HashBlock(block)
	fmt.Println(hex.EncodeToString(hash))
	assert.Equal(t, 32, len(hash))
}
