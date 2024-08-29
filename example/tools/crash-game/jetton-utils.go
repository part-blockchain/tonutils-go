package main

import (
	"context"
	"encoding/hex"
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
	"strconv"
	"strings"
)

// genWalletByMnemonicWords Generate Wallet based on mnemonic words
func genWalletByMnemonicWords(api wallet.TonAPI, seeds string, version wallet.Version) (error, *wallet.Wallet) {
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

// newJettonMasterClient 创建Jetton Master Client
func newJettonMasterClient(jettonMinterAddr string) (error, *context.Context, *jetton.Client) {
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

// DeployJettonMinter 部署Jetton Minter合约
func DeployJettonMinter(jettonMinterCodeFile, jettonWalletCodeFile string) error {
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
	fmt.Printf(GetScanCfg()+"%s\n", addr.String())
	// 更新jetton minter地址
	cfg.Jetton.JettonMinterAddr = addr.String()
	if err = UpdateGlobalCfg(cfg); nil != err {
		return err
	}
	return nil
}

// GetJettonData 获取Jetton Token信息
func GetJettonData(jettonMinterAddr string) (error, *jetton.Data) {
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

// GetJettonWallet 获取Jetton Wallet Data信息
func GetJettonWallet(jettonMinterAddr, ownerAddr string) error {
	if "" == jettonMinterAddr {
		// read from config file
		cfg, err := GetGlobalCfg()
		if nil != err {
			return err
		}
		jettonMinterAddr = cfg.Jetton.JettonMinterAddr
	}
	err, pCtx, master := newJettonMasterClient(jettonMinterAddr)
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

	// 获取jetton minter合约的content
	err, metadata := GetJettonMetaData()
	if nil != err {
		return err
	}
	decimals, err := strconv.Atoi(metadata.Decimals)
	if nil != err {
		decimals = 6
	}
	log.Println("jetton minter address:", jettonWalletData.JettonMinterAddr)
	log.Println("jetton wallet owner address:", jettonWalletData.OwnerAddr)
	log.Println("jetton wallet address:", tokenWallet.Address().String())
	log.Println("jetton balance:", tlb.MustFromNano(jettonWalletData.Balance, decimals))

	return nil
}

// MintToken 铸造Token
func MintToken(jettonMinterAddr, receiveAddr, amount string) error {
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
	txHash := ""
	if err, txHash = master.MintToken(pCtx, w, receiveAddr, amount, jettonDecimals, nil); err != nil {
		log.Fatal(err)
	}
	//log.Printf(GetScanCfg()+"%s\n", receiveAddr)
	log.Printf(GetScanCfg()+"/transaction/%s\n", txHash)
	return nil
}

func TransferToken(transferWallet *wallet.Wallet, jettonMinterAddr, receiveAddr, responseAddr, amount, comment, forwardGasFee string, payloadForward *cell.Cell) error {
	errMsg := ""
	if "" == receiveAddr {
		errMsg = fmt.Sprintf("receive address is empty")
		fmt.Println(errMsg)
		return errors.New(errMsg)
	}

	// read from config file
	cfg, err := GetGlobalCfg()
	if nil != err {
		return err
	}
	// 获取jetton精度
	jettonDecimals, _ := strconv.Atoi(cfg.Jetton.MetaData.Decimals)
	transferAmount := tlb.MustFromDecimal(amount, jettonDecimals)
	if "" == amount || "0" == transferAmount.String() {
		errMsg = fmt.Sprintf("transfer amount is empty or zero!")
		fmt.Println(errMsg)
		return errors.New(errMsg)
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
	log.Println("start to transfer jetton token...")

	if nil == transferWallet {
		// 导入Admin钱包
		err, transferWallet = genWalletByMnemonicWords(TonAPI, Seeds, WalletVersion)
		if err != nil || nil == transferWallet {
			errMsg := fmt.Sprintf("generate wallet by seed words failed: %s", err.Error())
			log.Println(errMsg)
			return errors.New(errMsg)
		}
	}

	// 获取jetton minter client对象
	err, pCtx, master := newJettonMasterClient(jettonMinterAddr)
	if err != nil || nil == master || nil == pCtx {
		return errors.New("new jetton master client failed")
	}
	// 获取jetton wallet对象, 获取from地址的jetton wallet对象
	fromAddr := transferWallet.WalletAddress()
	tokenWallet, err := master.GetJettonWallet(*pCtx, fromAddr)
	if err != nil || tokenWallet == nil {
		errMsg = "get jetton wallet failed."
		log.Fatal(errMsg)
	}
	// 查询from的jetton余额
	tokenBalance, err := tokenWallet.GetBalance(*pCtx)
	if err != nil {
		errMsg := fmt.Sprintf("get jetton wallet balance failed: %s", err.Error())
		log.Fatal(errMsg)
	}
	// 转账前的token余额
	coinBalance := tlb.MustFromNano(tokenBalance, jettonDecimals)
	log.Printf("jetton minter address:%s\n", jettonMinterAddr)
	log.Printf("from address:%s, jetton balance:%s, transfer amount:%s\n", fromAddr.String(), coinBalance.String(), transferAmount.String())

	// 判断是否有足够的token余额
	cmp := transferAmount.Cmp(coinBalance)
	if cmp > 0 {
		// token余额不足
		errMsg = fmt.Sprintf("balance:[%s] > transfer amount:[%s]", coinBalance.String(), transferAmount.String())
		log.Println(errMsg)
		return errors.New(errMsg)
	}

	// 转账jetton token附加参数
	if payloadForward == nil && comment != "" {
		payloadForward, err = wallet.CreateCommentCell(comment)
		if err != nil || payloadForward == nil {
			errMsg = fmt.Sprintf("create comment cell failed: %s", err.Error())
			log.Fatal(errMsg)
			return errors.New(errMsg)
		}
	}

	// address of receiver's wallet (not token wallet, just usual)
	to := address.MustParseAddr(receiveAddr)
	// 响应地址,一般为from地址
	if responseAddr == "" {
		responseAddr = fromAddr.String()
	}
	responseTo := address.MustParseAddr(responseAddr)
	if forwardGasFee == "" {
		forwardGasFee = "0.05"
	}
	forwardGasFeeCoin := tlb.MustFromTON(forwardGasFee)
	// 构建转账参数
	transferPayload, err := tokenWallet.BuildTransferPayloadV2(to, responseTo, transferAmount, forwardGasFeeCoin, payloadForward, nil)
	if err != nil {
		log.Fatal(err)
	}

	// your TON balance must be > 0.05 to send
	_, gasFee := forwardGasFeeCoin.Add(tlb.MustFromTON("0.05"))
	msg := wallet.SimpleMessage(tokenWallet.Address(), gasFee, transferPayload)

	log.Println("sending transfer jetton token transaction...")
	tx, _, err := transferWallet.SendWaitTransaction(*pCtx, msg)
	if err != nil {
		panic(err)
	}
	txHash := hex.EncodeToString(tx.Hash)
	// log.Println("transaction confirmed, hash:", txHash)
	log.Printf(GetScanCfg()+"/transaction/%s\n", txHash)
	return nil
}
