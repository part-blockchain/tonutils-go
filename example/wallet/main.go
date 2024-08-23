/*
钱包工具：
1.生成地址：
（1）.生成新地址
（2）.根据助记词生成地址
（3）.根据私钥生成地址

2.钱包操作：
（1）查询钱包余额
（2）转账
*/
package main

import (
	"flag"
)

// 生成地址类型
var genAddrType = flag.Int("gen_addr_type", 0, "generate address type:"+
	"[0: generate new address, 1: generate address by mnemonic words, 2: generate address by private key]")

// to地址
var isTransfer = flag.Bool("is_transfer", false, "is transfer")
var toAddr = flag.String("to_addr", "0QB8_1jtzFEA3LIUznSTQtHkp0HhJegU94l5fMEpT5qAJEXX", "receive token address")
var amount = flag.String("amount", "0.01", "transfer amount")

func init() {
	flag.Parse()
}

func main() {
	///////////////////////////// address operation ///////////////////////
	if -1 != *genAddrType {
		AddrOperation(*genAddrType)
	}

	// 转账操作
	if *isTransfer {
		TransferTon(*toAddr, *amount, "this is transfer test")
	}

}
