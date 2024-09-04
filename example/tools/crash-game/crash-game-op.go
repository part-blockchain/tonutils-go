/*
Crash Game合约操作：
1.部署Crash Game合约
2.获取Crash Game合约信息
3.创建新的一轮游戏
4.玩家下注
5.获取game wallet信息：玩家地址，下注信息，crash倍数等
6.管理员crash游戏
7.获取游戏记录信息：包括游戏轮数，游戏状态，随机数种子，crash倍数，玩家数量等
8.玩家结算游戏
9.获取玩家信息：包括玩家下注信息、游戏钱包，jetton余额等
10.管理员从资金池赎回token
*/
package main

import (
	"errors"
	"flag"
	"log"
)

// CrashGameOpType CrashGame操作类型
var gameOpType = flag.Int("game_op_type", -1, "crash game operation type:"+
	"[0: deploy crash game, 1: get crash game info, 2: new round for crash game, 3: bet for crash game,"+
	"4: get game wallet info, 5: crash game, 6: get game record info, 7: player settlement, 8: get player info]")

// deploy crash game
var crashGameCodeFile = flag.String("crash_game_code_file", "", "crash game code file path")
var gameRecordCodeFile = flag.String("game_record_code_file", "", "game record code file path")
var gameWalletCodeFile = flag.String("game_wallet_code_file", "", "game wallet code file path")

// crash game合约地址
var crashGameAddr = flag.String("crash_game_addr", "", "crash game contract address")

// 获取Crash Game合约信息
var showCode = flag.Bool("show_code", false, "show contract code")

// 玩家下注
var playerWalletIndex = flag.Int("player_wallet_index", 0, "player wallet index, "+
	"0: admin address, 1: player 1 address, 2: player 2 address, ...")
var betAmount = flag.String("bet_amount", "", "bet amount")
var betMultiple = flag.Uint64("bet_multiple", 0, "bet multiple")
var playerAddr = flag.String("player_addr", "", "player address")

// crash游戏
var roundNum = flag.Uint64("round_num", 0, "round number")

// 结算
var settleAddr = flag.String("settle_addr", "", "Settlement user address\n")

func CrashGameOperation() {
	if *gameOpType != -1 {
		err := errors.New("")
		switch *gameOpType {
		case 0:
			err = DeployCrashGame(*jettonMinterAddr, *jettonWalletCodeFile, *gameWalletCodeFile, *gameRecordCodeFile, *crashGameCodeFile)
		case 1:
			err, _ = GetCrashGameInfo(*crashGameAddr, *showCode)
		case 2:
			err = NewRound(*crashGameAddr)
		case 3:
			err = Bet(*playerWalletIndex, *crashGameAddr, *betAmount, *betMultiple)
		case 4:
			err = GetGameWalletInfo(*playerWalletIndex, *crashGameAddr, *playerAddr, *showCode)
		case 5:
			err = Crash(*crashGameAddr, *roundNum)
		case 6:
			err, _ = GetGameRecordInfo(*crashGameAddr, *roundNum, *showCode)
		case 7:
			err = Settlement(*playerWalletIndex, *crashGameAddr, *settleAddr, *roundNum)
		default:
			// do nothing
			err = errors.New("invalid crash game operation type")
		}

		if err != nil && "" != err.Error() {
			log.Fatal(err)
		}
	}
}
