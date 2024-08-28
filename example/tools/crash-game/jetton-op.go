/*
Jetton合约操作：指定jetton操作类型以及对应的操作参数
1.部署Jetton Minter合约
2.铸币
3.转账Jetton
4.查询Jetton余额
5.获取Jetton信息
*/

package main

import (
	"flag"
	"fmt"
)

// jetton操作类型
var jettonOpType = flag.Int("jetton_op_type", -1, "jetton operation type:"+
	"[0: deploy jetton minter, 1: get jetton data, 2: get jetton wallet data, 3: Mint Token,"+
	"4: Transfer Token, 5: Get Token Balance, 6: Get Token Info]")

// deploy jetton minter
var jettonMinterCodeFile = flag.String("jetton_minter_code_file", "", "Jetton minter code file path")
var jettonWalletCodeFile = flag.String("jetton_wallet_code_file", "", "Jetton wallet code file path")

// jetton minnter地址
// var jettonMinterAddr = flag.String("jetton_minter_addr", "EQBg-WGLMoQoA52la9C-i5wnQyNelVdZiE3j6ithX7bfz0MV", "Jetton Minter address")
var jettonMinterAddr = flag.String("jetton_minter_addr", "", "Jetton Minter address")
var jettonWalletOwner = flag.String("owner_addr", "0QB8_1jtzFEA3LIUznSTQtHkp0HhJegU94l5fMEpT5qAJEXX", "Jetton wallet owner address")

// Mint Token
// 铸币时，token的接收地址，为空时默认为owner地址
var mintReceiveAddr = flag.String("mint_receive_addr", "", "Mint Jetton token receive address")

// Transfer Token
// 转账时，token的接收地址
var transferReceiveAddr = flag.String("transfer_receive_addr", "0QB8_1jtzFEA3LIUznSTQtHkp0HhJegU94l5fMEpT5qAJEXX", "tranfer Jetton token receive address")
var comment = flag.String("comment", "", "Transfer Jetton token comment")

// Mint or Transfer Jetton token amount
var amount = flag.String("amount", "0.000001", "Mint or Transfer Jetton token amount")

func JettonOperation() {
	if *jettonOpType != -1 {
		switch *jettonOpType {
		case 0:
			DeployJettonMinter(*jettonMinterCodeFile, *jettonWalletCodeFile)
		case 1:
			GetJettonData(*jettonMinterAddr)
		case 2:
			GetJettonWallet(*jettonMinterAddr, *jettonWalletOwner)
		case 3:
			MintToken(*jettonMinterAddr, *mintReceiveAddr, *amount)
		case 4:
			TransferToken(*jettonMinterAddr, *transferReceiveAddr, *amount, *comment)
		default:
			// do nothing
			fmt.Println("Invalid jetton operation type")
		}
	}
}
