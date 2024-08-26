/*
Jetton合约操作：
1.部署Jetton Minter合约
2.铸币
3.转账Jetton
4.查询Jetton余额
5.获取Jetton信息
*/

package main

import (
	"flag"
)

// 操作类型
var opType = flag.Int("op_type", 0, "jetton operation type:"+
	"[0: deploy jetton minter, 1: get jetton data, 2: get jetton wallet data]")

// deploy jetton minter
var jettonMinterCodeFile = flag.String("jetton_minter_code_file", "", "Jetton minter code file path")
var jettonWalletCodeFile = flag.String("jetton_wallet_code_file", "", "Jetton wallet code file path")

// jetton minnter地址
// var jettonMinterAddr = flag.String("jetton_minter_addr", "EQBg-WGLMoQoA52la9C-i5wnQyNelVdZiE3j6ithX7bfz0MV", "Jetton Minter address")
var jettonMinterAddr = flag.String("jetton_minter_addr", "kQD__iZD2sqdQ22Xj94JUBSr2PXUNcAqUQQ9wkLezME-OsS2", "Jetton Minter address")
var jettonWalletOwner = flag.String("owner_addr", "0QB8_1jtzFEA3LIUznSTQtHkp0HhJegU94l5fMEpT5qAJEXX", "Jetton wallet owner address")

func init() {
	flag.Parse()
}

func main() {
	if *opType == 0 {
		DeployJettonMinter(*jettonMinterCodeFile, *jettonWalletCodeFile)
	} else if *opType == 1 {
		GetJettonData(*jettonMinterAddr)
	} else if *opType == 2 {
		GetJettonWallet(*jettonMinterAddr, *jettonWalletOwner)
	}
}
