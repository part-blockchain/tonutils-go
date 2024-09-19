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
	"math/big"
	"os"
	"strconv"
	"time"
)

const (
	OpBet      = 0x9b0663d8
	DictKeyLen = 16 // 游戏信息字典key长度
)

// 获取玩家的钱包
func getWalletByIndex(api wallet.TonAPI, walletIndex int, version wallet.Version) *wallet.Wallet {
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
		log.Fatalf("get [%s] code failed:%s\n", fileName, err)
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

func initCrashGameData(maxRoundsParallel uint64) *cell.Dictionary {
	dictGameInfos := cell.NewDict(DictKeyLen)
	for i := 0; i < int(maxRoundsParallel); i++ {
		k := big.NewInt(int64(i))
		gameInfo := &CrashGame.GameInfoPeerRound{
			RoundIndex:    uint64(i),     // 游戏轮数索引
			RoundNum:      uint64(i + 1), // 游戏轮数
			GameState:     1,             // 游戏状态: 0-bet; 1-游戏结束/未启动
			Seed:          0,             // 随机数种子
			CrashMultiple: 0,             // Crash乘数, 即爆炸倍数, 单位:%
			PlayerNums:    0,             // 玩家数量
			StartUnixTime: 0,             // 游戏开始时间(unix)
			StartTxTime:   0,             // 游戏开始交易时间(unix)
			StartBlkTime:  0,             // 游戏开始区块时间(unix)
		}
		v := cell.BeginCell().
			MustStoreUInt(gameInfo.RoundIndex, 32).
			MustStoreUInt(gameInfo.RoundNum, 32).
			MustStoreUInt(gameInfo.GameState, 32).
			MustStoreUInt(gameInfo.Seed, 32).
			MustStoreUInt(gameInfo.CrashMultiple, 32).
			MustStoreUInt(gameInfo.PlayerNums, 32).
			MustStoreUInt(gameInfo.StartUnixTime, 32).
			MustStoreUInt(gameInfo.StartTxTime, 64).
			MustStoreUInt(gameInfo.StartBlkTime, 64).
			EndCell()

		err := dictGameInfos.SetIntKey(k, v)
		if err != nil {
			log.Fatal(err)
		}
	}
	log.Printf("init crash game peer round hash:%v\n", hex.EncodeToString(dictGameInfos.AsCell().Hash()))
	return dictGameInfos
}

// getDeployCrashGameData 获取部署CrashGame的data
func getDeployCrashGameData(jettonMinterAddr, adminAddr *address.Address, maxRoundsParallel, minIntervalTime uint64,
	jettonWalletCode *cell.Cell, gameWalletCode *cell.Cell, gameRecordCode *cell.Cell) (_ *cell.Cell, err error) {

	// 部署合约初始化数据
	initData := &CrashGame.Data{
		CurrentRoundNum:   0,                 // 当前轮数, 用于统计游戏的总轮数
		MaxRoundsParallel: maxRoundsParallel, //  最大并行游戏轮数
		MinIntervalTime:   minIntervalTime,   //  一轮游戏从创建完成到crash的最小时间间隔,单位秒
		AdminAddr:         adminAddr,         //  管理员地址
		JettonMinterAddr:  jettonMinterAddr,  //  JettonMinter合约地址
		JettonWalletCode:  jettonWalletCode,  //  JettonWallet合约代码
		GameWalletCode:    gameWalletCode,    //  GameWallet合约代码
		GameRecordCode:    gameRecordCode,    //  GameRecord合约代码
	}

	data := cell.BeginCell().
		// MustStoreDict(initCrashGameData(maxRoundsParallel)).
		MustStoreDict(cell.NewDict(DictKeyLen)).
		MustStoreUInt(initData.CurrentRoundNum, 32).
		MustStoreUInt(initData.MaxRoundsParallel, 32).
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

// newGameWalletClient 创建GameWallet合约对象 Client
func newGameWalletClient(gameWalletAddr *address.Address) (error, *context.Context, *CrashGame.GameWalletClient) {
	if nil == TonAPI {
		TonAPI = GetTonAPIIns()
		if nil == TonAPI {
			return errors.New("get ton api instance failed"), nil, nil
		}
	}
	client := TonAPI.Client()
	// bound all requests to single ton node
	ctx := client.StickyContext(context.Background())
	gameWalletClient := CrashGame.NewGameWalletClient(TonAPI, gameWalletAddr)
	return nil, &ctx, gameWalletClient
}

// newGameRecordClient 创建GameRecord合约对象 Client
func newGameRecordClient(gameRecordAddr *address.Address) (error, *context.Context, *CrashGame.GameRecordClient) {
	if nil == TonAPI {
		TonAPI = GetTonAPIIns()
		if nil == TonAPI {
			return errors.New("get ton api instance failed"), nil, nil
		}
	}
	client := TonAPI.Client()
	// bound all requests to single ton node
	ctx := client.StickyContext(context.Background())
	gameRecordClient := CrashGame.NewGameRecordClient(TonAPI, gameRecordAddr)
	return nil, &ctx, gameRecordClient
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
	cfg, err := GetParamsCfg()
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
		cfg.CrashGameCfg.MaxRoundsParallel, cfg.CrashGameCfg.MinIntervalTime,
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
	addr, tx, _, err := w.DeployContractWaitTransaction(context.Background(), gasFee,
		msgBody, crashGameCode, deployData)
	if err != nil {
		panic(err)
	}
	// 浏览器展示部署合约的地址
	fmt.Printf(GetScanCfg()+"%s\n", addr.String())
	fmt.Printf(GetScanCfg()+"transaction/%s\n", hex.EncodeToString(tx.Hash))
	// 更新crash game地址
	cfg.CrashGameCfg.ContractAddr = addr.String()
	if err = UpdateParamsCfg(cfg); nil != err {
		return err
	}
	return nil
}

// 解析字典信息
func parseGameInfo(cellGameInfo *cell.Cell) *CrashGame.GameInfoPeerRound {
	// 遍历游戏信息
	gameInfo := CrashGame.GameInfoPeerRound{}
	lv := cellGameInfo.BeginParse()
	gameInfo.RoundIndex = lv.MustLoadUInt(32)    //  游戏轮数索引
	gameInfo.RoundNum = lv.MustLoadUInt(32)      //  游戏轮数
	gameInfo.GameState = lv.MustLoadUInt(32)     //  游戏状态: 0-bet; 1-游戏结束/未启动
	gameInfo.Seed = lv.MustLoadUInt(32)          //  随机数种子
	gameInfo.CrashMultiple = lv.MustLoadUInt(32) // Crash乘数, 即爆炸倍数, 单位:%
	gameInfo.PlayerNums = lv.MustLoadUInt(32)    // 玩家数量
	gameInfo.StartUnixTime = lv.MustLoadUInt(32) // 游戏开始时间(unix)
	gameInfo.StartTxTime = lv.MustLoadUInt(64)   // 游戏开始交易时间(unix)
	gameInfo.StartBlkTime = lv.MustLoadUInt(64)  // 游戏开始区块时间(unix)
	return &gameInfo
}

func parseGameInfoCell(cellGameInfo *cell.Cell, maxRoundsParallel uint64) []CrashGame.GameInfoPeerRound {
	dictGameInfos, err := cellGameInfo.BeginParse().ToDict(256)
	if err != nil {
		log.Fatal(err)
	}
	// 遍历游戏信息
	var gameInfos []CrashGame.GameInfoPeerRound
	for i := 0; i < int(maxRoundsParallel); i++ {
		gameInfo := CrashGame.GameInfoPeerRound{}
		k := big.NewInt(int64(i))
		// 获取key1对应的value
		lv, err := dictGameInfos.LoadValueByIntKey(k)
		if err != nil {
			log.Fatal(err)
		}
		// 获取value中的数据
		sl := lv.MustToCell().BeginParse()
		gameInfo.RoundIndex = sl.MustLoadUInt(32)    //  游戏轮数索引
		gameInfo.RoundNum = sl.MustLoadUInt(32)      //  游戏轮数
		gameInfo.GameState = sl.MustLoadUInt(32)     //  游戏状态: 0-bet; 1-游戏结束/未启动
		gameInfo.Seed = sl.MustLoadUInt(32)          //  随机数种子
		gameInfo.CrashMultiple = sl.MustLoadUInt(32) // Crash乘数, 即爆炸倍数, 单位:%
		gameInfo.PlayerNums = sl.MustLoadUInt(32)    // 玩家数量
		gameInfo.StartUnixTime = sl.MustLoadUInt(32) // 游戏开始时间(unix)
		gameInfo.StartTxTime = sl.MustLoadUInt(64)   // 游戏开始交易时间(unix)
		gameInfo.StartBlkTime = sl.MustLoadUInt(64)  // 游戏开始区块时间(unix)
		gameInfos = append(gameInfos, gameInfo)
	}
	return gameInfos
}

// GetCrashGameInfo 获取CrashGame信息
func GetCrashGameInfo(crashGameAddr string, showCode bool, roundNum uint64) (error, *CrashGame.CrashGameInfo) {
	// read from config file
	cfg, err := GetParamsCfg()
	if nil != err {
		return err, nil
	}
	if "" == crashGameAddr {
		crashGameAddr = cfg.CrashGameCfg.ContractAddr
	}

	if 0 == roundNum {
		roundNum = cfg.CrashGameCfg.RoundNum
	}

	err, ctx, crashGame := newCrashGameClient(crashGameAddr)
	if err != nil || nil == crashGame || nil == ctx {
		return errors.New("new crash game client failed"), nil
	}
	// CrashGame合约信息
	info := CrashGame.CrashGameInfo{}
	info.ContractAddr = address.MustParseAddr(crashGameAddr)
	// 获取合约数据
	data, err := crashGame.GetCrashGameData(*ctx, showCode, roundNum)
	if err != nil || nil == data {
		log.Fatal(err)
	}

	// CrashGame合约数据
	info.Data = data
	info.JettonWalletInfo.JettonMinterAddr = address.MustParseAddr(cfg.Jetton.JettonMinterAddr)

	// 获取crash game合约的jetton wallet信息
	// 获取jetton minter client对象
	err, pCtx, master := newJettonMasterClient(cfg.Jetton.JettonMinterAddr)
	if err != nil || nil == master || nil == pCtx {
		return errors.New("new jetton master client failed"), nil
	}
	tokenWallet, err := master.GetJettonWallet(*pCtx, info.ContractAddr)
	if err != nil || tokenWallet == nil {
		errMsg := "get jetton wallet failed."
		log.Fatal(errMsg)
	}

	// 查询from的jetton余额
	tokenBalance, err := tokenWallet.GetBalance(*pCtx)
	if err != nil {
		errMsg := fmt.Sprintf("get jetton wallet balance failed: %s", err.Error())
		log.Fatal(errMsg)
	}
	// 转账前的token余额
	jettonDecimals, _ := strconv.Atoi(cfg.Jetton.MetaData.Decimals)
	coinBalance := tlb.MustFromNano(tokenBalance, jettonDecimals)
	info.JettonWalletInfo.JettonWalletAddr = tokenWallet.Address()
	info.JettonWalletInfo.Balance = coinBalance.Val()

	// 获取crash game合约的每一轮游戏信息
	if data.CellGameInfo != nil {
		// 解析游戏信息
		// gameInfos := parseGameInfoCell(data.CellGameInfo, data.MaxRoundsParallel)
		gameInfo := parseGameInfo(data.CellGameInfo)
		//byteGameInfo, _ := json.Marshal(gameInfos)
		byteGameInfo, _ := json.MarshalIndent(gameInfo, "", "    ")
		log.Println("gameInfo:\n", string(byteGameInfo))

		//// 获取游戏记录地址
		//for _, gameInfo := range gameInfos {
		//	gameRecordAddr, err := crashGame.GetGameRecordAddr(*pCtx, gameInfo.RoundIndex)
		//	if err != nil || nil == gameRecordAddr {
		//		errMsg := fmt.Sprintf("get game record address failed: %s", err.Error())
		//		return errors.New(errMsg), nil
		//	}
		//	gameRecordInfo := CrashGame.GameRecordInfo{
		//		ContractAddr: gameRecordAddr,
		//	}
		//
		//	err, recordData := GetGameRecordInfo(crashGameAddr, gameInfo.RoundIndex, showCode)
		//	if err != nil || recordData == nil {
		//		//errMsg := fmt.Sprintf("get game record info failed: %s", err.Error())
		//		//log.Fatal(errMsg)
		//	}
		//	gameRecordInfo.Data = recordData
		//	info.GameRecordInfos = append(info.GameRecordInfos, gameRecordInfo)
		//}

	}

	// json格式化输出
	byData, _ := json.MarshalIndent(info, "", "    ")
	log.Printf("crash game info:\n%v\n", string(byData))
	return nil, &info
}

// GetCrashGameData 获取CrashGame合约数据
func GetCrashGameData(crashGameAddr string, showCode bool, roundNum uint64) (error, *CrashGame.Data) {
	if "" == crashGameAddr {
		// read from config file
		cfg, err := GetParamsCfg()
		if nil != err {
			return err, nil
		}
		crashGameAddr = cfg.CrashGameCfg.ContractAddr
	}
	err, ctx, crashGame := newCrashGameClient(crashGameAddr)
	if err != nil || nil == crashGame || nil == ctx {
		return errors.New("new crash game client failed"), nil
	}
	data, err := crashGame.GetCrashGameData(*ctx, showCode, roundNum)
	if err != nil || nil == data {
		errMsg := "get crash game data failed"
		log.Fatal(errMsg)
	}
	// json格式化输出
	//byData, _ := json.MarshalIndent(data, "", "    ")
	//log.Printf("crash game data:\n%v\n", string(byData))
	return nil, data
}

// NewRound 创建新的一轮游戏
func NewRound(crashGameAddr string) error {
	// read from config file
	cfg, err := GetParamsCfg()
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
	// 等待交易确认，新的round number上链
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
	if err = UpdateParamsCfg(cfg); nil != err {
		return err
	}
	return nil
}

// Bet 玩家下注
func Bet(playerWalletIndex int, crashGameAddr, betAmount string, betMultiple uint64, roundNum uint64) error {
	cfg, w := prepareBaseEnv(playerWalletIndex)
	if w == nil {
		return errors.New("generate player wallet by seed words failed")
	}
	if "" == crashGameAddr {
		crashGameAddr = cfg.CrashGameCfg.ContractAddr
	}

	if 0 == roundNum {
		roundNum = cfg.CrashGameCfg.RoundNum
	}

	if betAmount == "" {
		betAmount = cfg.CrashGameCfg.Bet.Amount
	}
	if betMultiple == 0 {
		betMultiple = cfg.CrashGameCfg.Bet.Multiple
	}

	// 转账jetton token到crash game合约地址
	// 从crash game合约中获取jetton minter地址
	err, data := GetCrashGameData(crashGameAddr, false, roundNum)
	if err != nil || nil == data {
		return errors.New("get crash game data failed")
	}
	// 下注的payload:组装转账的payload转发消息, 在crash-game合约的transfer_notification中处理下注信息
	betPayload := cell.BeginCell().
		MustStoreUInt(OpBet, 32).
		MustStoreUInt(roundNum, 32).
		MustStoreUInt(betMultiple, 32).
		EndCell()

	log.Printf("betPayLoad hash:%s\n", hex.EncodeToString(betPayload.Hash()))
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

// GetGameWalletInfo 获取game wallet信息
func GetGameWalletInfo(playerWalletIndex int, crashGameAddr, playerAddr string, showCode bool) error {
	cfg, w := prepareBaseEnv(playerWalletIndex) // admin wallet index
	// 玩家地址
	if playerAddr == "" {
		playerAddr = w.WalletAddress().String()
	}
	if "" == crashGameAddr {
		crashGameAddr = cfg.CrashGameCfg.ContractAddr
	}
	log.Printf("crash game address:%s, player address:%s\n", crashGameAddr, playerAddr)

	err, ctx, crashGame := newCrashGameClient(crashGameAddr)
	if err != nil || nil == crashGame || nil == ctx {
		return errors.New("new crash game client failed")
	}
	// 获取玩家的游戏钱包地址
	gameWalletAddr, err := crashGame.GetUserGameWalletAddr(*ctx, address.MustParseAddr(playerAddr))
	if err != nil || nil == gameWalletAddr {
		return errors.New("get user game wallet address failed")
	}

	// 构造game wallet合约客户端
	err, ctx, gameWalletClient := newGameWalletClient(gameWalletAddr)
	if err != nil || nil == crashGame || nil == ctx {
		return errors.New("new game wallet client failed")
	}

	// 获取game wallet信息
	data, err := gameWalletClient.GetGameWalletData(*ctx, showCode)
	if err != nil || nil == data {
		errMsg := fmt.Sprintf("get game wallet data failed: %s", err.Error())
		return errors.New(errMsg)
	}
	// json格式化输出
	byData, _ := json.MarshalIndent(data, "", "    ")
	log.Printf("game wallet data:\n%v\n", string(byData))
	return nil
}

// Crash Crash游戏
func Crash(crashGameAddr string, roundNum uint64) error {
	cfg, w := prepareBaseEnv(0) // admin wallet index
	if "" == crashGameAddr {
		crashGameAddr = cfg.CrashGameCfg.ContractAddr
	}
	if 0 == roundNum {
		roundNum = cfg.CrashGameCfg.RoundNum
	}
	// 获取jetton minter client对象
	err, pCtx, crashGame := newCrashGameClient(crashGameAddr)
	if err != nil || nil == crashGame || nil == pCtx {
		return errors.New("new crash game client failed")
	}
	log.Println("start to crash game...")

	txHash := ""
	// 计算gas fee
	betGasFee := 0.05
	forwardGasFee := 0.05
	gasFee := fmt.Sprintf("%f", betGasFee+forwardGasFee)
	gasFeeCoin := tlb.MustFromTON(gasFee)
	forwardGasFeeCoin := tlb.MustFromTON(fmt.Sprintf("%f", forwardGasFee))

	if err, txHash = crashGame.Crash(pCtx, w, roundNum, gasFeeCoin, forwardGasFeeCoin); err != nil {
		log.Fatal(err)
	}
	log.Printf(GetScanCfg()+"transaction/%s\n", txHash)
	return nil
}

// GetGameRecordInfo 获取游戏记录信息
func GetGameRecordInfo(crashGameAddr string, roundNum uint64, showCode bool) (error, *CrashGame.GameRecordData) {
	cfg, _ := prepareBaseEnv(-1)
	if "" == crashGameAddr {
		crashGameAddr = cfg.CrashGameCfg.ContractAddr
	}
	if 0 == roundNum {
		roundNum = cfg.CrashGameCfg.RoundNum
	}
	// 获取crash game client对象
	err, pCtx, crashGame := newCrashGameClient(crashGameAddr)
	if err != nil || nil == crashGame || nil == pCtx {
		errMsg := fmt.Sprintf("new crash game client failed: %s", err.Error())
		return errors.New(errMsg), nil
	}

	// 获取游戏记录地址
	gameRecordAddr, err := crashGame.GetGameRecordAddr(*pCtx, roundNum)
	if err != nil || nil == gameRecordAddr {
		errMsg := fmt.Sprintf("get game record address failed: %s", err.Error())
		return errors.New(errMsg), nil
	}

	// 构造game record合约客户端
	err, pCtx, gameRecordClient := newGameRecordClient(gameRecordAddr)
	if err != nil || nil == gameRecordClient || nil == pCtx {
		return errors.New("new game record client failed"), nil
	}

	log.Println("start to get game record info...")
	// 获取游戏记录信息
	data, err := gameRecordClient.GetGameRecordData(*pCtx, showCode)
	if err != nil || nil == data {
		errMsg := fmt.Sprintf("get game record info failed: %s", err.Error())
		log.Println(errMsg)
		return errors.New(errMsg), nil
	}
	// json格式化输出
	byData, _ := json.MarshalIndent(data, "", "    ")
	log.Printf("game record info:\n%v\n", string(byData))
	return nil, data
}

// Settlement 玩家结算游戏
func Settlement(playerWalletIndex int, crashGameAddr, settleAddr string, roundNum uint64) error {
	cfg, w := prepareBaseEnv(playerWalletIndex)
	if w == nil {
		return errors.New("generate player wallet by seed words failed")
	}
	if "" == crashGameAddr {
		crashGameAddr = cfg.CrashGameCfg.ContractAddr
	}

	if "" == settleAddr {
		settleAddr = w.WalletAddress().String()
	}

	if 0 == roundNum {
		roundNum = cfg.CrashGameCfg.RoundNum
	}

	sender := w.WalletAddress()
	log.Printf("settlement info: {crash game address:%s, sender address:%s, settle user address:%s, roundNum:%d}\n",
		crashGameAddr, sender.String(), settleAddr, roundNum)

	err, ctx, crashGame := newCrashGameClient(crashGameAddr)
	if err != nil || nil == crashGame || nil == ctx {
		return errors.New("new crash game client failed")
	}
	// 计算玩家的游戏钱包地址
	gameWalletAddr, err := crashGame.GetUserGameWalletAddr(*ctx, address.MustParseAddr(settleAddr))
	if err != nil || nil == gameWalletAddr {
		return errors.New("get user game wallet address failed")
	}
	log.Println("game wallet address:", gameWalletAddr.String())

	// 构造game wallet合约客户端
	err, ctx, gameWalletClient := newGameWalletClient(gameWalletAddr)
	if err != nil || nil == crashGame || nil == ctx {
		return errors.New("new game wallet client failed")
	}

	txHash := ""
	// 计算gas fee
	betGasFee := 0.05
	jettonForwardGasFee := 0.1
	forwardGasFee := 2*betGasFee + jettonForwardGasFee
	forwardGasFeeCoin := tlb.MustFromTON(fmt.Sprintf("%f", forwardGasFee))
	gasFee := fmt.Sprintf("%f", betGasFee+forwardGasFee)
	gasFeeCoin := tlb.MustFromTON(gasFee)

	// 调用结算方法
	if err, txHash = gameWalletClient.Settlement(ctx, w, roundNum, cfg.CrashGameCfg.MaxRoundsParallel,
		gasFeeCoin, forwardGasFeeCoin, address.MustParseAddr(settleAddr)); err != nil {
		log.Fatal(err)
	}
	log.Printf(GetScanCfg()+"transaction/%s\n", txHash)
	return nil
}

func getJettonWalletInfo(jettonMinterAddr, ownerAddr string) (error, *CrashGame.JettonWalletInfo) {
	// 获取jetton minter client对象
	err, pCtx, master := newJettonMasterClient(jettonMinterAddr)
	if err != nil || nil == master || nil == pCtx {
		return errors.New("new jetton master client failed"), nil
	}
	// 获取jetton minter信息
	tokenWallet, err := master.GetJettonWallet(*pCtx, address.MustParseAddr(ownerAddr))
	if err != nil || tokenWallet == nil {
		errMsg := "get jetton wallet failed."
		log.Fatal(errMsg)
	}

	// 查询jetton余额
	tokenBalance, err := tokenWallet.GetBalance(*pCtx)
	if err != nil {
		errMsg := fmt.Sprintf("get jetton wallet balance failed: %s", err.Error())
		log.Fatal(errMsg)
	}
	jettonWalletInfo := &CrashGame.JettonWalletInfo{
		JettonMinterAddr: address.MustParseAddr(jettonMinterAddr),
		JettonWalletAddr: tokenWallet.Address(),
		Balance:          tokenBalance,
	}

	return nil, jettonWalletInfo
}

// GetPlayerInfo 获取玩家记录信息
func GetPlayerInfo(playerWalletIndex int, crashGameAddr, playerAddr string, showCode bool) error {
	cfg, w := prepareBaseEnv(playerWalletIndex)
	// 玩家地址
	if playerAddr == "" {
		playerAddr = w.WalletAddress().String()
	}
	if "" == crashGameAddr {
		crashGameAddr = cfg.CrashGameCfg.ContractAddr
	}
	playerInfo := CrashGame.PlayerInfo{}
	// 玩家地址
	playerInfo.PlayerAddr = address.MustParseAddr(playerAddr)
	// 获取crash game合约的jetton wallet信息
	err, jettonWalletInfo := getJettonWalletInfo(cfg.Jetton.JettonMinterAddr, playerAddr)
	jettonDecimals, _ := strconv.Atoi(cfg.Jetton.MetaData.Decimals)
	playerInfo.JettonWalletInfo.JettonMinterAddr = jettonWalletInfo.JettonMinterAddr
	playerInfo.JettonWalletInfo.JettonWalletAddr = jettonWalletInfo.JettonWalletAddr
	playerInfo.JettonWalletInfo.Balance = tlb.MustFromNano(jettonWalletInfo.Balance, jettonDecimals).Val()
	log.Printf("crash game address:%s, player address:%s\n", crashGameAddr, playerAddr)

	err, ctx, crashGame := newCrashGameClient(crashGameAddr)
	if err != nil || nil == crashGame || nil == ctx {
		return errors.New("new crash game client failed")
	}
	// 获取玩家的游戏钱包地址
	gameWalletAddr, err := crashGame.GetUserGameWalletAddr(*ctx, address.MustParseAddr(playerAddr))
	if err != nil || nil == gameWalletAddr {
		return errors.New("get user game wallet address failed")
	}
	// game wallet address
	playerInfo.GameWalletInfo.ContractAddr = gameWalletAddr

	// 构造game wallet合约客户端
	err, ctx, gameWalletClient := newGameWalletClient(gameWalletAddr)
	if err != nil || nil == crashGame || nil == ctx {
		return errors.New("new game wallet client failed")
	}

	// 获取game wallet信息
	data, err := gameWalletClient.GetGameWalletData(*ctx, showCode)
	if err != nil || nil == data {
		errMsg := fmt.Sprintf("get game wallet data failed: %s", err.Error())
		return errors.New(errMsg)
	}
	playerInfo.GameWalletInfo.Data = data
	// json格式化输出
	byData, _ := json.MarshalIndent(playerInfo, "", "    ")
	log.Printf("game wallet data:\n%v\n", string(byData))
	return nil
}
