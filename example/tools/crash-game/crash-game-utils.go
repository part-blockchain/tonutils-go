package main

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/xssnick/tonutils-go/address"
	"github.com/xssnick/tonutils-go/tlb"
	CrashGame "github.com/xssnick/tonutils-go/ton/crash-game"
	"github.com/xssnick/tonutils-go/ton/wallet"
	"github.com/xssnick/tonutils-go/tvm/cell"
	"io/ioutil"
	"log"
	"os"
	"time"
)

const (
	OpBet = 0x9b0663d8
)

// 获取玩家的钱包
func getPlayerWallet(api wallet.TonAPI, walletIndex int, version wallet.Version) *wallet.Wallet {
	// 获取玩家钱包
	if walletIndex >= len(PlayerSeeds) {
		log.Fatal("player wallet index out of range")
		return nil
	}
	err, w := genWalletByMnemonicWords(api, PlayerSeeds[walletIndex], version)
	if err != nil || nil == w {
		log.Fatal("generate wallet by seed words failed:", err.Error())
	}
	log.Printf("use player wallet[%d]: %s\n", walletIndex, w.WalletAddress().String())
	return w
}

// 获取合约代码的code
func getContractCode(contractCodeFilePath, fileName string) *cell.Cell {
	if "" == contractCodeFilePath {
		dir, _ := os.Getwd()
		contractCodeFilePath = fmt.Sprintf("%s/example/tools/contracts/build/%s.cell", dir, fileName)
	}
	fmt.Printf("%s code file path: %s\n", fileName, contractCodeFilePath)
	data, err := ioutil.ReadFile(contractCodeFilePath)
	if err != nil {
		fmt.Printf("get %s code failed:%s\n", fileName, err)
		return nil
	}
	// 读取文件内容
	codeCell, err := cell.FromBOC(data)
	if err != nil {
		fmt.Printf("get %s code FromBOC failed:%s\n", fileName, err)
		panic(err)
	}

	return codeCell
}

// 获取crash game code
func getCrashGameCode(crashGameCodeFilePath string) *cell.Cell {
	if "" == crashGameCodeFilePath {
		dir, _ := os.Getwd()
		crashGameCodeFilePath = fmt.Sprintf("%s/example/tools/contracts/build/crash-game.cell", dir)
	}
	fmt.Printf("crash game code file path: %s\n", crashGameCodeFilePath)
	data, err := ioutil.ReadFile(crashGameCodeFilePath)
	if err != nil {
		fmt.Println("get crash game code failed:", err)
		return nil
	}
	// 读取文件内容
	codeCell, err := cell.FromBOC(data)
	if err != nil {
		fmt.Println("get crash game code failed:", err)
		panic(err)
	}

	return codeCell
}

// 获取crash game wallet code
func getGameWalletCode(gameWalletCodeFilePath string) *cell.Cell {
	if "" == gameWalletCodeFilePath {
		dir, _ := os.Getwd()
		gameWalletCodeFilePath = fmt.Sprintf("%s/example/tools/contracts/build/game-wallet.cell", dir)
	}
	fmt.Printf("game wallet code file path: %s\n", gameWalletCodeFilePath)
	data, err := ioutil.ReadFile(gameWalletCodeFilePath)
	if err != nil {
		fmt.Println("get jetton wallet code failed:", err)
		return nil
	}
	// 读取文件内容
	codeCell, err := cell.FromBOC(data)
	if err != nil {
		fmt.Println("get jetton wallet code failed:", err)
		panic(err)
	}

	return codeCell
}

// 获取crash game record code
func getGameRecordCode(jettonWalletCodeFilePath string) *cell.Cell {
	if "" == jettonWalletCodeFilePath {
		dir, _ := os.Getwd()
		jettonWalletCodeFilePath = fmt.Sprintf("%s/example/tools/contracts/build/jetton-wallet.cell", dir)
	}
	fmt.Printf("jetton wallet code file path: %s\n", jettonWalletCodeFilePath)
	data, err := ioutil.ReadFile(jettonWalletCodeFilePath)
	if err != nil {
		fmt.Println("get jetton wallet code failed:", err)
		return nil
	}
	// 读取文件内容
	codeCell, err := cell.FromBOC(data)
	if err != nil {
		fmt.Println("get jetton wallet code failed:", err)
		panic(err)
	}

	return codeCell
}

// getDeployCrashGameData 获取部署CrashGame的data
func getDeployCrashGameData(jettonMinterAddr *address.Address, adminAddr *address.Address,
	jettonWalletCode *cell.Cell, gameWalletCode *cell.Cell, gameRecordCode *cell.Cell) (_ *cell.Cell, err error) {

	// 部署合约初始化数据
	initData := &CrashGame.Data{
		RoundNum:         0,                //  游戏轮数
		GameState:        1,                //  游戏状态: 0-bet; 1-游戏结束/未启动
		Seed:             0,                //  随机数种子
		CrashMultiple:    0,                // Crash乘数, 即爆炸倍数, 单位:%
		PlayerNums:       0,                // 玩家数量
		StartUnixTime:    0,                // 游戏开始时间(unix)
		StartTxTime:      0,                // 游戏开始交易时间(unix)
		StartBlkTime:     0,                // 游戏开始区块时间(unix)
		MinIntervalTime:  0,                //  一轮游戏从创建完成到crash的最小时间间隔,单位秒
		AdminAddr:        adminAddr,        //  管理员地址
		JettonMinterAddr: jettonMinterAddr, //  JettonMinter合约地址
		JettonWalletCode: jettonWalletCode, //  JettonWallet合约代码
		GameWalletCode:   gameWalletCode,   //  GameWallet合约代码
		GameRecordCode:   gameRecordCode,   //  GameRecord合约代码
	}

	data := cell.BeginCell().
		MustStoreUInt(initData.RoundNum, 32).
		MustStoreUInt(initData.GameState, 32).
		MustStoreUInt(initData.Seed, 32).
		MustStoreUInt(initData.CrashMultiple, 32).
		MustStoreUInt(initData.PlayerNums, 32).
		MustStoreUInt(initData.StartUnixTime, 32).
		MustStoreUInt(initData.StartTxTime, 64).
		MustStoreUInt(initData.StartBlkTime, 64).
		MustStoreUInt(initData.MinIntervalTime, 32).
		MustStoreAddr(initData.AdminAddr).
		MustStoreAddr(initData.JettonMinterAddr).
		MustStoreRef(initData.JettonWalletCode).
		MustStoreRef(initData.GameWalletCode).
		MustStoreRef(initData.GameRecordCode).
		EndCell()

	// 校验data hash
	fmt.Println("deploy crash game initialize data hash:", hex.EncodeToString(data.Hash()))
	return data, nil
}

// newCrashGameClient 创建CrashGame合约对象 Client
func newCrashGameClient(crashGameAddr string) (error, *context.Context, *CrashGame.Client) {
	if nil == TonAPI {
		TonAPI = GetTonAPIIns()
		if nil == TonAPI {
			return errors.New("get ton api instance failed"), nil, nil
		}
	}
	client := TonAPI.Client()
	// bound all requests to single ton node
	ctx := client.StickyContext(context.Background())
	crashGameContract := address.MustParseAddr(crashGameAddr)
	crashGameClient := CrashGame.NewCrashGameClient(TonAPI, crashGameContract)
	return nil, &ctx, crashGameClient
}

// DeployCrashGame 部署CrashGame合约
func DeployCrashGame(jettonMinterAddr, jettonWalletCodeFile, gameWalletCodeFile, gameRecordCodeFile, crashGameCodeFile string) error {
	if nil == TonAPI {
		TonAPI = GetTonAPIIns()
		if nil == TonAPI {
			return errors.New("get ton api instance failed")
		}
	}
	// 获取jetton全局配置
	cfg, err := GetGlobalCfg()
	if nil != err {
		return err
	}
	if jettonMinterAddr == "" {
		jettonMinterAddr = cfg.Jetton.JettonMinterAddr
	}
	// 生成钱包
	err, w := genWalletByMnemonicWords(TonAPI, Seeds, WalletVersion)
	if err != nil || nil == w {
		return errors.New("generate wallet by seed words failed")
	}

	log.Println("Deploy wallet:", w.WalletAddress().String())
	// 生成部署合约数据（初始化合约数据）
	deployData, err := getDeployCrashGameData(address.MustParseAddr(jettonMinterAddr), w.WalletAddress(),
		getContractCode(jettonWalletCodeFile, "jetton-wallet"), getContractCode(gameWalletCodeFile, "game-wallet"),
		getContractCode(gameRecordCodeFile, "game-record"))

	if nil != err {
		return errors.New("get Deploy crash game Data failed")
	}
	msgBody := cell.BeginCell().EndCell()
	gasFee := tlb.MustFromTON("0.2")
	crashGameCode := getContractCode(crashGameCodeFile, "crash-game")

	netName := "test network"
	if *IsMainNet {
		netName = "main network"
	}
	fmt.Printf("Deploying crash game contract to ton %s...\n", netName)
	addr, _, _, err := w.DeployContractWaitTransaction(context.Background(), gasFee,
		msgBody, crashGameCode, deployData)
	if err != nil {
		panic(err)
	}
	// 浏览器展示部署合约的地址
	fmt.Printf(GetScanCfg()+"%s\n", addr.String())
	// 更新crash game地址
	cfg.CrashGameCfg.ContractAddr = addr.String()
	if err = UpdateGlobalCfg(cfg); nil != err {
		return err
	}
	return nil
}

// GetCrashGameData 获取CrashGame信息
func GetCrashGameData(crashGameAddr string, showCode bool) (error, *CrashGame.Data) {
	if "" == crashGameAddr {
		// read from config file
		cfg, err := GetGlobalCfg()
		if nil != err {
			return err, nil
		}
		crashGameAddr = cfg.CrashGameCfg.ContractAddr
	}
	err, ctx, crashGame := newCrashGameClient(crashGameAddr)
	if err != nil || nil == crashGame || nil == ctx {
		return errors.New("new crash game client failed"), nil
	}
	data, err := crashGame.GetCrashGameData(*ctx, showCode)
	if err != nil || nil == data {
		errMsg := "get crash game data failed"
		log.Fatal(errMsg)
	}
	// json格式化输出
	byData, _ := json.MarshalIndent(data, "", "    ")
	log.Printf("crash game data:\n%v\n", string(byData))
	return nil, data
}

// NewRound 创建新的一轮游戏
func NewRound(crashGameAddr string) error {
	// read from config file
	cfg, err := GetGlobalCfg()
	if nil != err {
		return err
	}
	if "" == crashGameAddr {
		crashGameAddr = cfg.CrashGameCfg.ContractAddr
	}
	if nil == TonAPI {
		TonAPI = GetTonAPIIns()
		if nil == TonAPI {
			return errors.New("get ton api instance failed")
		}
	}

	err, w := genWalletByMnemonicWords(TonAPI, Seeds, WalletVersion)
	if err != nil || nil == w {
		errMsg := fmt.Sprintf("generate wallet by seed words failed: %s", err.Error())
		log.Println(errMsg)
		return errors.New(errMsg)
	}
	// 获取jetton minter client对象
	err, pCtx, crashGame := newCrashGameClient(crashGameAddr)
	if err != nil || nil == crashGame || nil == pCtx {
		return errors.New("new crash game client failed")
	}
	// 获取旧的round number
	beforeRoundNum, err := crashGame.GetRoundNum(*pCtx)
	if err != nil {
		return err
	}
	log.Println("start to new round for crash game...")
	txHash := ""
	if err, txHash = crashGame.NewRound(pCtx, w); err != nil {
		log.Fatal(err)
	}
	afterRoundNum, err := crashGame.GetRoundNum(*pCtx)
	if err != nil {
		return err
	}
	for {
		if afterRoundNum > beforeRoundNum {
			break
		}
		// 休眠2秒
		time.Sleep(2 * time.Second)
		afterRoundNum, _ = crashGame.GetRoundNum(*pCtx)
	}
	log.Printf(GetScanCfg()+"transaction/%s\n", txHash)
	// 更新round number
	cfg.CrashGameCfg.RoundNum = afterRoundNum
	if err = UpdateGlobalCfg(cfg); nil != err {
		return err
	}
	return nil
}

// Bet 玩家下注
func Bet(playerWalletIndex int, crashGameAddr, betAmount string, betMultiple uint64) error {
	if nil == TonAPI {
		TonAPI = GetTonAPIIns()
		if nil == TonAPI {
			return errors.New("get ton api instance failed")
		}
	}
	// 获取玩家钱包
	w := getPlayerWallet(TonAPI, playerWalletIndex, WalletVersion)
	if nil == w {
		return errors.New("generate wallet by seed words failed")
	}
	// read from config file
	cfg, err := GetGlobalCfg()
	if nil != err {
		return err
	}
	if "" == crashGameAddr {
		crashGameAddr = cfg.CrashGameCfg.ContractAddr
	}
	if betAmount == "" {
		betAmount = cfg.CrashGameCfg.Bet.Amount
	}
	if betMultiple == 0 {
		betMultiple = cfg.CrashGameCfg.Bet.Multiple
	}

	// 转账jetton token到crash game合约地址
	// 从crash game合约中获取jetton minter地址
	err, data := GetCrashGameData(crashGameAddr, false)
	if err != nil || nil == data {
		return errors.New("get crash game data failed")
	}
	// 下注的payload:组装转账的payload转发消息, 在crash-game合约的transfer_notification中处理下注信息
	betPayload := cell.BeginCell().
		MustStoreUInt(OpBet, 32).
		MustStoreUInt(data.RoundNum, 32).
		MustStoreUInt(betMultiple, 32).
		EndCell()
	// 计算gas fee
	betGasFee := 0.05
	notifyGasFee := 0.05
	forwardGasFee := fmt.Sprintf("%f", betGasFee+notifyGasFee)
	if err = TransferToken(w, data.JettonMinterAddr.String(), crashGameAddr, w.WalletAddress().String(), betAmount, "", forwardGasFee, betPayload); nil != err {
		errMsg := fmt.Sprintf("bet failed, transfer token failed: %s", err.Error())
		log.Println(errMsg)
		return errors.New(errMsg)
	}
	// 获取crash game信息
	return nil
}
