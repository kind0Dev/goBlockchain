package main

import (
	"context"
	"log"
	"time"

	"github.com/djeday123/blockchain2/crypto"
	"github.com/djeday123/blockchain2/node"
	"github.com/djeday123/blockchain2/proto"
	"github.com/djeday123/blockchain2/util"
	"google.golang.org/grpc"
)

func main() {
	//node := node.NewNode()
	makeNode(":3000", []string{}, true)
	time.Sleep(time.Second)
	makeNode(":4000", []string{":3000"}, false)
	time.Sleep(time.Second)
	makeNode(":5000", []string{":4000"}, false)

	for {
		time.Sleep(time.Millisecond * 100)
		makeTransaction()
	}
	// go func() {
	// 	for {
	// 		time.Sleep(4 * time.Second)
	// 	}
	// }()
	// select {}
}

func makeNode(listenAddr string, bootstrapNodes []string, isValidator bool) *node.Node {
	cfg := node.ServerConfig{
		Version:    "bl-0.1",
		ListenAddr: listenAddr,
	}
	if isValidator {
		cfg.PrivateKey = crypto.GeneratePrivateKey()
	}

	n := node.NewNode(cfg)
	go n.Start(listenAddr, bootstrapNodes)
	// if len(bootstrapNodes) > 0 {
	// 	if err := n.BootstrapNetwork(bootstrapNodes); err != nil {
	// 		log.Fatal(err)
	// 	}
	// }

	return n
}

func makeTransaction() {

	client, err := grpc.Dial(":3000", grpc.WithInsecure())
	//client, err := grpc.Dial(":3000", grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatal(err)
	}

	c := proto.NewNodeClient(client)
	privKey := crypto.GeneratePrivateKey()
	// version := &proto.Version{
	// 	Version:    "bl-0.1",
	// 	Height:     1,
	// 	ListenAddr: ":4000",
	// }

	tx := &proto.Transaction{
		Version: 1,
		Inputs: []*proto.TxInput{
			{
				PrevTxHash:   util.RandomHash(),
				PrevOutIndex: 0,
				PublicKey:    privKey.Public().Bytes(),
			},
		},
		Outputs: []*proto.TxOutput{
			{
				Amount:  99,
				Address: privKey.Public().Address().Bytes(),
			},
		},
	}

	_, err = c.HandleTransaction(context.TODO(), tx)
	if err != nil {
		log.Fatal(err)
	}
}
