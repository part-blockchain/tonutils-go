/*
Crash Game合约操作：
1.部署Crash Game合约
2.获取Crash Game合约信息
*/
package main

import (
	"flag"
	"fmt"
)

// CrashGameOpType CrashGame操作类型
var gameOpType = flag.Int("game_op_type", -1, "crash game operation type:"+
	"[0: deploy crash game, 1: get crash game info]")

// deploy crash game
var crashGameCodeFile = flag.String("crash_game_code_file", "", "crash game code file path")
var gameRecordCodeFile = flag.String("game_record_code_file", "", "game record code file path")
var gameWalletCodeFile = flag.String("game_wallet_code_file", "", "game wallet code file path")

// crash game合约地址
var crashGameAddr = flag.String("crash_game_addr", "", "crash game contract address")

func CrashGameOperation() {
	if *gameOpType != -1 {
		switch *gameOpType {
		case 0:
			DeployCrashGame(*jettonMinterAddr, *jettonWalletCodeFile, *gameWalletCodeFile, *gameRecordCodeFile, *crashGameCodeFile)
		default:
			// do nothing
			fmt.Println("Invalid crash game operation type")
		}
	}
}
