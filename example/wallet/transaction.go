/*
普通交易相关操作
*/
package main

import (
	"context"
	"encoding/base64"
	"errors"
	"github.com/xssnick/tonutils-go/address"
	"github.com/xssnick/tonutils-go/tlb"
	"log"
)

// TransferTon 转移TON
func TransferTon(to, amount, comment string) (err error) {
	if nil == TonAPI {
		TonAPI = GetTonAPIIns()
		if nil == TonAPI {
			return errors.New("get ton api instance failed")
		}
	}
	client := TonAPI.Client()
	// bound all requests to single ton node
	ctx := client.StickyContext(context.Background())
	err, w := GenWalletByMnemonicWords(TonAPI, Seeds, WalletVersion)
	if err != nil {
		log.Fatalln("FromSeed err:", err.Error())
		return
	}

	log.Println("wallet address:", w.WalletAddress())

	log.Println("fetching and checking proofs since config init block, it may take near a minute...")
	block, err := TonAPI.CurrentMasterchainInfo(context.Background())
	if err != nil {
		log.Fatalln("get masterchain info err: ", err.Error())
		return
	}
	log.Println("master proof checks are completed successfully, now communication is 100% safe!")

	balance, err := w.GetBalance(ctx, block)
	if err != nil {
		log.Fatalln("GetBalance err:", err.Error())
		return
	}
	coinAmount := tlb.MustFromTON(amount)
	cmp := balance.Cmp(coinAmount)
	if cmp >= 0 {
		addr := address.MustParseAddr(to)

		log.Println("sending transaction and waiting for confirmation...")

		// if destination wallet is not initialized (or you don't care)
		// you should set bounce to false to not get money back.
		// If bounce is true, money will be returned in case of not initialized destination wallet or smart-contract error
		bounce := false
		transfer, err := w.BuildTransfer(addr, tlb.MustFromTON(amount), bounce, comment)
		if err != nil {
			log.Fatalln("Transfer err:", err.Error())
			return
		}

		tx, block, err := w.SendWaitTransaction(ctx, transfer)
		if err != nil {
			log.Fatalln("SendWaitTransaction err:", err.Error())
			return
		}

		balance, err = w.GetBalance(ctx, block)
		if err != nil {
			log.Fatalln("GetBalance err:", err.Error())
			return
		}

		log.Printf("transaction confirmed at block %d, hash: %s balance left: %s", block.SeqNo,
			base64.StdEncoding.EncodeToString(tx.Hash), balance.String())

		return
	}
	log.Println("not enough balance:", balance.String())
	err = errors.New("not enough balance")
	return
}
