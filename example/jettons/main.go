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
	"[0: get jetton data, 1: get jetton wallet data]")

// jetton minnter地址
var jettonMinterAddr = flag.String("jetton_minter_addr", "EQBg-WGLMoQoA52la9C-i5wnQyNelVdZiE3j6ithX7bfz0MV", "Jetton Minter address")
var jettonWalletOwner = flag.String("owner_addr", "0QB8_1jtzFEA3LIUznSTQtHkp0HhJegU94l5fMEpT5qAJEXX", "Jetton wallet owner address")

func init() {
	flag.Parse()
}

func main() {
	if *opType == 0 {
		GetJettonData(*jettonMinterAddr)
	} else if *opType == 1 {
		GetJettonWallet(*jettonMinterAddr, *jettonWalletOwner)
	}
}
