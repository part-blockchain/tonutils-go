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
	"github.com/xssnick/tonutils-go/tvm/cell"
	"io/ioutil"
	"log"
	"os"
	"strconv"
)

//// CrashGameData crash game合约数据
//type CrashGameData struct {
//	RoundNum         uint64           //  游戏轮数
//	GameState        uint64           //  游戏状态: 0-bet; 1-游戏结束/未启动
//	Seed             uint64           //  随机数种子
//	CrashMultiple    uint64           // Crash乘数, 即爆炸倍数, 单位:%
//	PlayerNums       uint64           // 玩家数量
//	StartUnixTime    uint64           // 游戏开始时间(unix)
//	StartTxTime      uint64           // 游戏开始交易时间(unix)
//	StartBlkTime     uint64           // 游戏开始区块时间(unix)
//	MinIntervalTime  uint64           //  一轮游戏从创建完成到crash的最小时间间隔,单位秒
//	AdminAddr        *address.Address //  管理员地址
//	JettonMinterAddr *address.Address //  JettonMinter合约地址
//	JettonWalletCode *cell.Cell       //  JettonWallet合约代码
//	GameWalletCode   *cell.Cell       //  GameWallet合约代码
//	GameRecordCode   *cell.Cell       //  GameRecord合约代码
//}

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
	fmt.Printf(GetScanCfg(), addr.String())
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
func NewRound(jettonMinterAddr, receiveAddr, amount string) error {
	// read from config file
	cfg, err := GetGlobalCfg()
	if nil != err {
		return err
	}
	if "" == jettonMinterAddr {
		jettonMinterAddr = cfg.Jetton.JettonMinterAddr
	}
	if nil == TonAPI {
		TonAPI = GetTonAPIIns()
		if nil == TonAPI {
			return errors.New("get ton api instance failed")
		}
	}
	log.Println("start to mint jetton token...")
	err, w := genWalletByMnemonicWords(TonAPI, Seeds, WalletVersion)
	if err != nil || nil == w {
		errMsg := fmt.Sprintf("generate wallet by seed words failed: %s", err.Error())
		log.Println(errMsg)
		return errors.New(errMsg)
	}
	// 获取jetton minter client对象
	err, pCtx, master := newJettonMasterClient(jettonMinterAddr)
	if err != nil || nil == master || nil == pCtx {
		return errors.New("new jetton master client failed")
	}
	if "" == receiveAddr {
		// 默认接收地址为当前钱包地址，jetton minter合约的owner地址
		receiveAddr = w.WalletAddress().String()
	}
	// 铸币
	jettonDecimals, _ := strconv.Atoi(cfg.Jetton.MetaData.Decimals)
	if err, _ = master.MintToken(pCtx, w, receiveAddr, amount, jettonDecimals, nil); err != nil {
		log.Fatal(err)
	}
	log.Printf(GetScanCfg(), receiveAddr)
	return nil
}
