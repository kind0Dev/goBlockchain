package types

import (
	//"fmt"
	"testing"

	"github.com/djeday123/blockchain2/crypto"
	"github.com/djeday123/blockchain2/proto"
	"github.com/djeday123/blockchain2/util"
	"github.com/stretchr/testify/assert"
)

// my balance 100 coins

// want to send 5 coins to "AAA"

func TestNewTransaction(t *testing.T) {
	fromPrivKey := crypto.GeneratePrivateKey()
	fromAddress := fromPrivKey.Public().Address().Bytes()

	toPrivKey := crypto.GeneratePrivateKey()
	toAddress := toPrivKey.Public().Address().Bytes()

	input := &proto.TxInput{
		PrevTxHash:   util.RandomHash(),
		PrevOutIndex: 0,
		PublicKey:    fromPrivKey.Public().Bytes(),
	}

	output1 := &proto.TxOutput{
		Amount:  5,
		Address: toAddress,
	}

	output2 := &proto.TxOutput{
		Amount:  95,
		Address: fromAddress,
	}

	tx := &proto.Transaction{
		Version: 1,
		Inputs:  []*proto.TxInput{input},
		Outputs: []*proto.TxOutput{output1, output2},
	}
	sig := SignTransaction(fromPrivKey, tx)

	input.Signature = sig.Bytes()
	assert.True(t, VerifyTransaction(tx))
	//fmt.Printf("%+v\n", tx)
}
