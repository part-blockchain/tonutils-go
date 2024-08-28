package main

import (
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"github.com/xssnick/tonutils-go/address"
	"github.com/xssnick/tonutils-go/tlb"
	"github.com/xssnick/tonutils-go/ton/jetton"
	"github.com/xssnick/tonutils-go/tvm/cell"
	"io/ioutil"
	"log"
	"math/big"
	"os"
	"strconv"
)

// 获取crash game code
func getCrashGameCode(jettonWalletCodeFilePath string) *cell.Cell {
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
func getDeployCrashGameData(totalSupply *big.Int, owner *address.Address,
	content jetton.ContentAny, jettonWalletCode *cell.Cell) (_ *cell.Cell, err error) {

	conData, err := jetton.GenJettonContentCell(content)
	if err != nil {
		return nil, fmt.Errorf("failed to convert jetton minter content to cell: %w", err)
	}
	// 校验content hash
	fmt.Printf("jetton content cell hash: %s\n", hex.EncodeToString(conData.Hash()))
	data := cell.BeginCell().
		MustStoreCoins(totalSupply.Uint64()). // total supply, 提示：不要用MustStoreUInt(totalSupply.Uint64(), 64)
		MustStoreAddr(owner).
		MustStoreRef(conData).
		MustStoreRef(jettonWalletCode).
		EndCell()

	// 校验data hash
	fmt.Println("deploy jetton minter initialize data hash:", hex.EncodeToString(data.Hash()))
	if err != nil {
		return nil, fmt.Errorf("failed to convert depoly jetton minter data to cell: %w", err)
	}

	return data, nil
}

// newCrashGameClient 创建CrashGame合约对象 Client
func newCrashGameClient(jettonMinterAddr string) (error, *context.Context, *jetton.Client) {
	if nil == TonAPI {
		TonAPI = GetTonAPIIns()
		if nil == TonAPI {
			return errors.New("get ton api instance failed"), nil, nil
		}
	}
	client := TonAPI.Client()
	// bound all requests to single ton node
	ctx := client.StickyContext(context.Background())
	tokenContract := address.MustParseAddr(jettonMinterAddr)
	master := jetton.NewJettonMasterClient(TonAPI, tokenContract)
	return nil, &ctx, master
}

// DeployCrashGame 部署CrashGame合约
func DeployCrashGame(jettonMinterCodeFile, jettonWalletCodeFile string) error {
	if nil == TonAPI {
		TonAPI = GetTonAPIIns()
		if nil == TonAPI {
			return errors.New("get ton api instance failed")
		}
	}
	log.Println("Deploying jetton contract to ton blockchain...")
	err, w := genWalletByMnemonicWords(TonAPI, Seeds, WalletVersion)
	if err != nil || nil == w {
		return errors.New("generate wallet by seed words failed")
	}

	log.Println("Deploy wallet:", w.WalletAddress().String())
	fmt.Println("Deploying jetton minter contract to ton blockchain...")
	// 获取jetton全局配置
	cfg, err := GetGlobalCfg()
	if nil != err {
		return err
	}
	// 生成部署合约数据（初始化合约数据）
	deployData, err := getDeployJettonMinterData(big.NewInt(0), w.WalletAddress(), &cfg.Jetton.MetaData, getJettonWalletCode(jettonWalletCodeFile))
	if nil != err {
		return errors.New("get Deploy Jetton Minter Data failed")
	}
	msgBody := cell.BeginCell().EndCell()
	addr, _, _, err := w.DeployContractWaitTransaction(context.Background(), tlb.MustFromTON("0.05"),
		msgBody, getJettonMinterCode(jettonMinterCodeFile), deployData)
	if err != nil {
		panic(err)
	}
	// 浏览器展示部署合约的地址
	fmt.Printf(GetScanCfg(), addr.String())
	// 更新jetton minter地址
	cfg.Jetton.JettonMinterAddr = addr.String()
	if err = UpdateGlobalCfg(cfg); nil != err {
		return err
	}
	return nil
}

// GetCrashGameData 获取CrashGame信息
func GetCrashGameData(jettonMinterAddr string) (error, *jetton.Data) {
	if "" == jettonMinterAddr {
		// read from config file
		cfg, err := GetGlobalCfg()
		if nil != err {
			return err, nil
		}
		jettonMinterAddr = cfg.Jetton.JettonMinterAddr
	}
	err, ctx, master := newJettonMasterClient(jettonMinterAddr)
	if err != nil || nil == master || nil == ctx {
		return errors.New("new jetton master client failed"), nil
	}
	data, err := master.GetJettonData(*ctx)
	if err != nil {
		log.Fatal(err)
	}

	content := data.Content.(*jetton.MetaData)
	log.Println("total supply:", data.TotalSupply.Uint64())
	log.Println("mintable:", data.Mintable)
	log.Println("admin addr:", data.AdminAddr)
	log.Println("jetton content:")
	log.Println("	name:", content.Name)
	log.Println("	Description:", content.Description)
	log.Println("	Symbol:", content.Symbol)
	log.Println("	Decimals:", content.Decimals)
	log.Println("	Image:", content.Image)
	log.Println("	ImageData:", content.ImageData)
	log.Println("	URI:", content.URI) // 链下
	log.Println("	AmountStyle:", content.AmountStyle)
	log.Println("	RenderType:", content.RenderType)

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
