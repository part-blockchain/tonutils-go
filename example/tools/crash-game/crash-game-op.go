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

func CrashGameOperation() {
	if *gameOpType != -1 {
		switch *gameOpType {
		case 0:
			DeployJettonMinter(*jettonMinterCodeFile, *jettonWalletCodeFile)
		default:
			// do nothing
			fmt.Println("Invalid crash game operation type")
		}
	}
}
