/*
地址，钱包相关操作
*/
package main

import (
	"crypto/ed25519"
	"encoding/hex"
	"github.com/xssnick/tonutils-go/ton/wallet"
	"log"
	"strings"
)

// GenNewAddr 生成新地址
func GenNewAddr(version wallet.Version) {
	log.Println("generate new address...")

	words := wallet.NewSeed()
	log.Println("seed words:", strings.Join(words, " "))
	w, err := wallet.FromSeed(nil, words, version)
	if nil != err || nil == w {
		log.Fatalln("generate address failed by seed words:", err.Error())
		return
	}
	priKey := w.PrivateKey()
	log.Println("Wallet Version:", version.String())
	log.Println("priKey:", hex.EncodeToString(priKey))
	pubKey := w.PrivateKey().Public().(ed25519.PublicKey)
	log.Println("pubKey:", hex.EncodeToString(pubKey))
	log.Println("wallet address - BounceAble:", w.Address())
	log.Println("wallet address - Non-BounceAble:", w.WalletAddress())
}

func GenAddrByPriKey(priKeyHex string) {
	log.Println("generate address by private key...")
	// 将十六进制字符串转换为字节切片
	priKeyBytes, err := hex.DecodeString(priKeyHex)
	if err != nil {
		log.Fatalf("Failed to decode private key: %v", err)
	}

	// 在 ED25519 中，私钥通常是 64 字节，前 32 字节是种子
	if len(priKeyBytes) != 64 {
		log.Fatal("Invalid private key length")
	}

	priKey := ed25519.PrivateKey(priKeyBytes)
	log.Println("priKey:", hex.EncodeToString(priKey))
	// 从私钥中生成公钥（ed25519的私钥后32个字节数据是公钥）
	//pubKey := ed25519.PublicKey(priKeyBytes[32:])
	pubKey := priKey.Public().(ed25519.PublicKey)
	w, err := wallet.FromPrivateKey(nil, priKey, WalletVersion)
	if nil != err || nil == w {
		log.Fatalln("generate address failed by private key:", err.Error())
		return
	}
	log.Println("pubKey:", hex.EncodeToString(pubKey))
	log.Println("wallet address - BounceAble:", w.Address())
	log.Println("wallet address - Non-BounceAble:", w.WalletAddress())
}

// GenAddrByMnemonicWords reference http://tonhelloworld.com/01-wallet/
// Generate addresses based on mnemonic words
func GenAddrByMnemonicWords(seeds string, version wallet.Version, isBounceAble bool) (error, string) {
	log.Println("generate address by mnemonic words...")
	// seed words of account, you can generate them with any wallet or using wallet.NewSeed() method
	words := strings.Split(seeds, " ")
	w, err := wallet.FromSeed(nil, words, version)
	if err != nil {
		log.Fatalln("FromSeed err:", err.Error())
		return err, ""
	}

	priKey := w.PrivateKey()
	log.Println("Wallet Version:", version.String())
	log.Println("priKey:", hex.EncodeToString(priKey))
	pubKey := w.PrivateKey().Public().(ed25519.PublicKey)
	log.Println("pubKey:", hex.EncodeToString(pubKey))
	address := ""
	if isBounceAble {
		address = w.Address().String()
	} else {
		address = w.WalletAddress().String()
	}
	log.Println("wallet address - BounceAble:", w.Address())
	log.Println("wallet address - Non-BounceAble:", w.WalletAddress())
	return nil, address
}

// GenWalletByMnemonicWords Generate Wallet based on mnemonic words
func GenWalletByMnemonicWords(api wallet.TonAPI, seeds string, version wallet.Version) (error, *wallet.Wallet) {
	log.Println("generate address by mnemonic words...")
	// seed words of account, you can generate them with any wallet or using wallet.NewSeed() method
	words := strings.Split(seeds, " ")
	w, err := wallet.FromSeed(api, words, version)
	if err != nil {
		log.Fatalln("FromSeed err:", err.Error())
		return err, nil
	}

	return nil, w
}

// AddrOperation 地址操作
func AddrOperation(opType int) {
	if 0 == opType {
		GenNewAddr(WalletVersion)
	} else if 1 == opType {
		GenAddrByMnemonicWords(Seeds, WalletVersion, true)
	} else if 2 == opType {
		GenAddrByPriKey(PriKeyHex)
	} else if 3 == opType {
		GenWalletByMnemonicWords(nil, Seeds, WalletVersion)
	}
}
