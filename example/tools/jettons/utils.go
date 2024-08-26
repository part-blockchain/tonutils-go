package main

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/xssnick/tonutils-go/address"
	"github.com/xssnick/tonutils-go/tlb"
	"github.com/xssnick/tonutils-go/ton/jetton"
	"github.com/xssnick/tonutils-go/ton/wallet"
	"github.com/xssnick/tonutils-go/tvm/cell"
	"io/ioutil"
	"log"
	"math/big"
	"os"
	"strings"
)

// GenWalletByMnemonicWords Generate Wallet based on mnemonic words
func GenWalletByMnemonicWords(api wallet.TonAPI, seeds string, version wallet.Version) (error, *wallet.Wallet) {
	log.Println("generate address by mnemonic words...")
	// seed words of account, you can generate them with any wallet or using wallet.NewSeed() method
	words := strings.Split(seeds, " ")
	w, err := wallet.FromSeed(api, words, version)
	if err != nil {
		log.Fatalln("FromSeed err:", err.Error())
		return err, nil
	}

	return nil, w
}

// 获取jetton minter code
func getJettonMinterCode(jettonMinterCodeFilePath string) *cell.Cell {
	if "" == jettonMinterCodeFilePath {
		dir, _ := os.Getwd()
		jettonMinterCodeFilePath = fmt.Sprintf("%s/example/tools/contracts/build/jetton-minter.cell", dir)
	}
	fmt.Printf("jetton minter code file path: %s\n", jettonMinterCodeFilePath)
	data, err := ioutil.ReadFile(jettonMinterCodeFilePath)
	if err != nil {
		fmt.Println("get jetton minter code failed:", err)
		return nil
	}
	// 读取文件内容
	codeCell, err := cell.FromBOC(data)
	if err != nil {
		fmt.Println("get jetton minter code failed:", err)
		panic(err)
	}

	return codeCell
}

// 获取jetton wallet code
func getJettonWalletCode(jettonWalletCodeFilePath string) *cell.Cell {
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

// getDeployJettonMinterData 获取部署Jetton Minter的data
func getDeployJettonMinterData(totalSupply *big.Int, owner *address.Address,
	content jetton.ContentAny, jettonWalletCode *cell.Cell) (_ *cell.Cell, err error) {

	conData, err := jetton.GenJettonContentCell(content)
	if err != nil {
		return nil, fmt.Errorf("failed to convert jetton minter content to cell: %w", err)
	}
	fmt.Printf("jetton content cell hash: %s\n", hex.EncodeToString(conData.Hash()))
	data := cell.BeginCell().
		MustStoreUInt(totalSupply.Uint64(), 64).
		MustStoreAddr(owner).
		MustStoreRef(conData).
		MustStoreRef(jettonWalletCode).
		EndCell()

	if err != nil {
		return nil, fmt.Errorf("failed to convert depoly jetton minter data to cell: %w", err)
	}

	return data, nil
}

// DeployJettonMinter 部署Jetton Minter合约
func DeployJettonMinter(jettonMinterCodeFile, jettonWalletCodeFile string) error {
	if nil == TonAPI {
		TonAPI = GetTonAPIIns()
		if nil == TonAPI {
			return errors.New("get ton api instance failed")
		}
	}
	log.Println("Deploying jetton contract to ton blockchain...")
	err, w := GenWalletByMnemonicWords(TonAPI, Seeds, WalletVersion)
	if err != nil || nil == w {
		return errors.New("generate wallet by seed words failed")
	}

	log.Println("Deploy wallet:", w.WalletAddress().String())
	msgBody := cell.BeginCell().EndCell()

	fmt.Println("Deploying jetton minter contract to ton blockchain...")
	// 生成jetton minter合约的content
	content := jetton.MetaData{}
	if err = json.Unmarshal([]byte(JettonContentCfg), &content); nil != err {
		errMsg := fmt.Sprintf("unmarshal jetton content config failed:%v", err)
		fmt.Println(errMsg)
		return errors.New(errMsg)
	}
	// 生成部署合约数据（初始化合约数据）
	deployData, err := getDeployJettonMinterData(big.NewInt(0), w.WalletAddress(), &content, getJettonWalletCode(jettonWalletCodeFile))
	if nil != err {
		return errors.New("get Deploy Jetton Minter Data failed")
	}
	addr, _, _, err := w.DeployContractWaitTransaction(context.Background(), tlb.MustFromTON("0.05"),
		msgBody, getJettonMinterCode(jettonMinterCodeFile), deployData)
	if err != nil {
		panic(err)
	}

	fmt.Printf("Deployed contract: https://testnet.tonviewer.com/%s\n", addr.String())
	return nil
}

// NewJettonMasterClient 创建Jetton Master Client
func NewJettonMasterClient(jettonMinterAddr string) (error, *context.Context, *jetton.Client) {
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

// GetJettonData 获取Jetton Token信息
func GetJettonData(jettonMinterAddr string) (error, *jetton.Data) {
	err, ctx, master := NewJettonMasterClient(jettonMinterAddr)
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

// GetJettonWallet 获取Jetton Wallet Data信息
func GetJettonWallet(jettonMinterAddr, ownerAddr string) error {
	err, pCtx, master := NewJettonMasterClient(jettonMinterAddr)
	if err != nil || nil == master || nil == pCtx {
		return errors.New("new jetton master client failed")
	}
	ctx := *pCtx
	tokenWallet, err := master.GetJettonWallet(ctx, address.MustParseAddr(ownerAddr))
	if err != nil {
		log.Fatal(err)
	}
	// 获取jetton wallet data
	jettonWalletData, err := tokenWallet.GetWalletData(ctx)
	if err != nil {
		log.Fatal(err)
	}
	decimals := 6
	log.Println("jetton minter address:", jettonWalletData.JettonMinterAddr)
	log.Println("jetton wallet owner address:", jettonWalletData.OwnerAddr)
	log.Println("jetton balance:", tlb.MustFromNano(jettonWalletData.Balance, decimals))

	return nil
}
