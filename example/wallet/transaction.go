/*
普通交易相关操作
*/
package main

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/valyala/fasthttp"
	"github.com/xssnick/tonutils-go/address"
	"github.com/xssnick/tonutils-go/tlb"
	"log"
	"math"
	"strconv"
)

// TransferTon 转移TON
func TransferTon(to, amount, comment string) error {
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
		return err
	}

	log.Println("wallet address:", w.WalletAddress())

	log.Println("fetching and checking proofs since config init block, it may take near a minute...")
	block, err := TonAPI.CurrentMasterchainInfo(context.Background())
	if err != nil {
		log.Fatalln("get masterchain info err: ", err.Error())
		return err
	}
	log.Println("master proof checks are completed successfully, now communication is 100% safe!")

	balance, err := w.GetBalance(ctx, block)
	if err != nil {
		log.Fatalln("GetBalance err:", err.Error())
		return err
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
			return err
		}

		tx, block, err := w.SendWaitTransaction(ctx, transfer)
		if err != nil {
			log.Fatalln("SendWaitTransaction err:", err.Error())
			return err
		}

		balance, err = w.GetBalance(ctx, block)
		if err != nil {
			log.Fatalln("GetBalance err:", err.Error())
			return err
		}

		log.Printf("transaction confirmed at block %d, hash: %s balance left: %s", block.SeqNo,
			base64.StdEncoding.EncodeToString(tx.Hash), balance.String())

		return err
	}
	log.Println("not enough balance:", balance.String())
	err = errors.New("not enough balance")
	return err
}

// GetBalance 获取余额
func GetBalance(address string) (error, float64) {
	log.Println("get balance...")
	rpcUrl := GetRpcUrl()
	apiKey := GetApiKey()
	funcName := "getAddressBalance"
	url := fmt.Sprintf("%s/%s?address=%s&api_key=%s", rpcUrl, funcName, address, apiKey)

	statusCode, body, err := fasthttp.Get(nil, url)
	if err != nil {
		log.Fatalf("Error retrieving account balance: %v", err)
		return err, 0
	}
	if statusCode != fasthttp.StatusOK {
		errMsg := fmt.Sprintf("Unexpected status code: %d. Response body: %s", statusCode, body)
		log.Fatalf(errMsg)
		return errors.New(errMsg), 0
	}

	type Response struct {
		OK     bool   `json:"ok"`
		Result string `json:"result"`
	}
	res := Response{}
	if err := json.Unmarshal(body, &res); nil != err {
		log.Fatalf("json Unmarshal failed:%v", err)
		return err, 0
	}

	// log.Printf("Balance of %s is %s", address, res.Result)
	// 将字符串转换为 int64
	value, err := strconv.ParseInt(res.Result, 10, 64)
	if err != nil {
		fmt.Printf("Error parsing string: %v\n", err)
		return err, 0
	}

	// 计算10的n次方
	divisor := math.Pow(10, float64(TonDecimals))
	// 除以精度因子，并转换为 float64
	result := float64(value) / divisor

	// 输出结果
	// fmt.Printf("The value in float is: %f\n", result)
	log.Printf("Balance of %s is %f TON\n", address, result)
	return nil, result
}
