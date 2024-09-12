package main

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"github.com/xssnick/tonutils-go/address"
	"github.com/xssnick/tonutils-go/tlb"
	"github.com/xssnick/tonutils-go/ton"
	CrashGame "github.com/xssnick/tonutils-go/ton/crash-game"
	"github.com/xssnick/tonutils-go/ton/wallet"
	"github.com/xssnick/tonutils-go/tvm/cell"
	"log"
	"math/big"
	"testing"
)

var testDictCodeFile = flag.String("test_dict_code_file", "", "test dict code file path")

/*
func智能合约dict类型测试：
1. 部署合约
2. 获取合约信息
*/

// TestDictData test dict合约存储数据
type TestDictData struct {
	DictInfos *cell.Cell //  字典信息
}

// getDeployTestDictData 获取部署TestDict的data
func getDeployTestDictData() (_ *cell.Cell, err error) {
	dictGameInfos := cell.NewDict(32)
	k := big.NewInt(int64(0))
	gameInfo := &CrashGame.GameInfoPeerRound{
		RoundIndex: uint64(111), // 游戏轮数索引
		//RoundNum:      uint64(0 + 1), // 游戏轮数
		//GameState:     1,             // 游戏状态: 0-bet; 1-游戏结束/未启动
		//Seed:          0,             // 随机数种子
		//CrashMultiple: 0,             // Crash乘数, 即爆炸倍数, 单位:%
		//PlayerNums:    0,             // 玩家数量
		//StartUnixTime: 0,             // 游戏开始时间(unix)
		//StartTxTime:   0,             // 游戏开始交易时间(unix)
		//StartBlkTime:  0,             // 游戏开始区块时间(unix)
	}
	v := cell.BeginCell().
		MustStoreUInt(gameInfo.RoundIndex, 32).
		//MustStoreUInt(gameInfo.RoundNum, 32).
		//MustStoreUInt(gameInfo.GameState, 32).
		//MustStoreUInt(gameInfo.Seed, 32).
		//MustStoreUInt(gameInfo.CrashMultiple, 32).
		//MustStoreUInt(gameInfo.PlayerNums, 32).
		//MustStoreUInt(gameInfo.StartUnixTime, 32).
		//MustStoreUInt(gameInfo.StartTxTime, 64).
		//MustStoreUInt(gameInfo.StartBlkTime, 64).
		EndCell()

	err = dictGameInfos.SetIntKey(k, v)
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("test dict hash:%v\n", hex.EncodeToString(dictGameInfos.AsCell().Hash()))

	//data := cell.BeginCell().
	//	MustStoreDict(dictGameInfos).
	//	EndCell()

	data := cell.BeginCell().
		MustStoreDict(cell.NewDict(256)).
		EndCell()

	// 校验data hash
	fmt.Println("deploy test dict initialize data hash:", hex.EncodeToString(data.Hash()))
	return data, nil
}

// 从执行结果中获取对应的值
func getValueFromExecutionResult(res *ton.ExecutionResult, index *uint, keyName string, isAddrType bool) (value interface{}) {
	if res == nil || res.Length() <= *index {
		errMsg := fmt.Errorf("%s get err: invalid result or index:%d", keyName, index)
		log.Fatal(errMsg)
		return nil
	}
	typeName := res.Type(*index)
	if typeName == "null" {
		// 无效的null类型
		errMsg := fmt.Errorf("%s get err: invalid null type:%d", keyName, index)
		log.Fatal(errMsg)
		return nil
	}
	err := errors.New("")
	switch typeName {
	case "*big.Int":
		value, err = res.Int(*index)
	case "*cell.Slice":
		value, err = res.Slice(*index)
		if isAddrType && value != nil && err == nil {
			value, err = value.(*cell.Slice).LoadAddr()
		}
	case "*cell.Cell":
		value, err = res.Cell(*index)
	}

	if err != nil {
		errMsg := fmt.Errorf("%s get err: %v", keyName, err)
		log.Fatal(errMsg)
		return nil
	}
	*index += 1
	return value
}

func getKeyById(keyId uint64) []byte {
	keyCell := cell.BeginCell().
		MustStoreUInt(keyId, 32).
		EndCell()
	return keyCell.Hash()
}

func getBigIntKeyById(keyId uint64) *big.Int {

	keyCell := cell.BeginCell().
		MustStoreUInt(keyId, 32).
		EndCell()
	return big.NewInt(0).SetBytes(keyCell.Hash())
}

// 解析字典信息
func parseTestDictCell(cellDictInfos *cell.Cell) *CrashGame.GameInfoPeerRound {

	// dictInfos := cellDictInfos.AsDict(256)
	// 遍历游戏信息
	gameInfo := CrashGame.GameInfoPeerRound{}
	// k := big.NewInt(int64(0))
	// lv, err := dictInfos.LoadValueByIntKey(k)

	//k := getBigIntKeyById(0)
	//// 获取key1对应的value
	//lv, err := dictInfos.LoadValueByIntKey(k)
	//if err != nil {
	//	log.Fatal(err)
	//}
	// 获取value中的数据
	// sl := lv.MustToCell().BeginParse()
	lv := cellDictInfos.BeginParse()
	gameInfo.RoundIndex = lv.MustLoadUInt(32) //  游戏轮数索引
	return &gameInfo
}

// 将 *big.Int 转换为固定长度的字节切片
func bigIntToBytes(n *big.Int, length int) []byte {
	bytes := n.Bytes()
	if len(bytes) >= length {
		return bytes[len(bytes)-length:]
	}

	// 如果字节切片不足指定长度，前面填充零
	paddedBytes := make([]byte, length)
	copy(paddedBytes[length-len(bytes):], bytes)
	return paddedBytes
}

// DeployTestDict 部署TestDict合约
func DeployTestDict() error {
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
	// 生成钱包
	err, w := genWalletByMnemonicWords(TonAPI, Seeds, WalletVersion)
	if err != nil || nil == w {
		return errors.New("generate wallet by seed words failed")
	}

	log.Println("Deploy wallet:", w.WalletAddress().String())
	// 生成部署合约数据（初始化合约数据）
	deployData, err := getDeployTestDictData()

	if nil != err {
		return errors.New("get Deploy TestDict contract failed")
	}
	msgBody := cell.BeginCell().EndCell()
	gasFee := tlb.MustFromTON("0.02")
	crashGameCode := getContractCode(*testDictCodeFile, "dict-test")

	netName := "test network"
	if *IsMainNet {
		netName = "main network"
	}
	fmt.Printf("Deploying TestContract contract to ton %s...\n", netName)
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

// GetTestDictData 获取TestDict合约数据
func GetTestDictData(testDictAddr string) (error, *TestDictData) {
	if "" == testDictAddr {
		// read from config file
		cfg, err := GetParamsCfg()
		if nil != err {
			return err, nil
		}
		testDictAddr = cfg.CrashGameCfg.ContractAddr
	}

	if nil == TonAPI {
		TonAPI = GetTonAPIIns()
		if nil == TonAPI {
			return errors.New("get ton api instance failed"), nil
		}
	}
	client := TonAPI.Client()
	// bound all requests to single ton node
	ctx := client.StickyContext(context.Background())

	b, err := TonAPI.CurrentMasterchainInfo(ctx)
	if err != nil {
		return fmt.Errorf("failed to get masterchain info: %w", err), nil
	}
	keyId := 0
	res, err := TonAPI.WaitForBlock(b.SeqNo).RunGetMethod(ctx, b, address.MustParseAddr(testDictAddr), "get_info", keyId)
	if err != nil {
		errMsg := fmt.Errorf("failed to run get_info method by test dict contract: %w", err)
		log.Fatal(errMsg)
		return errMsg, nil
	}

	index := uint(0)
	data := &TestDictData{
		DictInfos: getValueFromExecutionResult(res, &index, "DictInfos", false).(*cell.Cell), // 字典所有信息
	}

	// 获取crash game合约的每一轮游戏信息
	if data.DictInfos != nil {
		// 解析游戏信息
		gameInfos := parseTestDictCell(data.DictInfos)
		//byGameInfos, _ := json.Marshal(gameInfos)
		byGameInfos, _ := json.MarshalIndent(gameInfos, "", "    ")
		log.Println("test dict info:\n", string(byGameInfos))
	}
	return nil, data
}

// GetKeyByID 调用合约接口获取指定key
func GetKeyByID(testDictAddr string, keyId uint64, t *testing.T) error {
	if "" == testDictAddr {
		// read from config file
		cfg, err := GetParamsCfg()
		if nil != err {
			return err
		}
		testDictAddr = cfg.CrashGameCfg.ContractAddr
	}

	if nil == TonAPI {
		TonAPI = GetTonAPIIns()
		if nil == TonAPI {
			return errors.New("get ton api instance failed")
		}
	}
	client := TonAPI.Client()
	// bound all requests to single ton node
	ctx := client.StickyContext(context.Background())

	b, err := TonAPI.CurrentMasterchainInfo(ctx)
	if err != nil {
		return fmt.Errorf("failed to get masterchain info: %w", err)
	}

	mothodName := "get_key_by_id"
	res, err := TonAPI.WaitForBlock(b.SeqNo).RunGetMethod(ctx, b, address.MustParseAddr(testDictAddr), mothodName, keyId)
	if err != nil {
		log.Fatalf("failed to run %s method by test dict contract: %v", mothodName, err)
	}
	index := uint(0)
	keyData := getValueFromExecutionResult(res, &index, "getKeyById", false).(*big.Int)

	// 转换为 32 字节（SHA256 输出长度）
	dataBytes := bigIntToBytes(keyData, 32)

	// 将哈希值转换为十六进制字符串
	fmt.Println("get key by id from contract:", hex.EncodeToString(dataBytes[:]))

	bytesKey := getKeyById(keyId)
	fmt.Println("key hash:", hex.EncodeToString(bytesKey))

	if !bytes.Equal(dataBytes, bytesKey) {
		t.Fatal("the key by id not equal")
		return nil
	}
	return nil
}

// SetTestDictPayload set test dict payload
type SetTestDictPayload struct {
	_          tlb.Magic  `tlb:"#20a4c01e"` // Set Test dict data opcode
	QueryID    uint64     `tlb:"## 64"`
	UpdateCell *cell.Cell `tlb:"maybe ^"`
}

const (
	OpSetTestDict = 0x20a4c01e
)

// SetDictData 设置TestDict合约数据
func SetDictData(ctx context.Context, contractAddr string, w *wallet.Wallet) (error, string) {

	buf := make([]byte, 8)
	if _, err := rand.Read(buf); err != nil {
		return err, ""
	}
	rnd := binary.LittleEndian.Uint64(buf)
	body := cell.BeginCell().
		MustStoreUInt(OpSetTestDict, 32).
		MustStoreUInt(rnd, 64).
		MustStoreRef(cell.BeginCell().MustStoreUInt(200, 32).EndCell()).
		EndCell()

	// your TON balance must be > 0.05 to send
	msg := wallet.SimpleMessage(address.MustParseAddr(contractAddr), tlb.MustFromTON("0.01"), body)

	log.Println("sending test dict transaction...")
	tx, _, err := w.SendWaitTransaction(ctx, msg)
	if err != nil {
		panic(err)
	}
	txHash := hex.EncodeToString(tx.Hash)
	// log.Println("transaction confirmed, hash:", txHash)
	return nil, txHash
}

// UpdateDictData 更新TestDict合约数据
func UpdateDictData(ctx context.Context, contractAddr string, w *wallet.Wallet) (error, string) {

	buf := make([]byte, 8)
	if _, err := rand.Read(buf); err != nil {
		return err, ""
	}
	rnd := binary.LittleEndian.Uint64(buf)

	body := cell.BeginCell().
		MustStoreUInt(OpSetTestDict, 32).
		MustStoreUInt(rnd, 64).
		MustStoreUInt(888, 32).
		EndCell()

	// your TON balance must be > 0.05 to send
	msg := wallet.SimpleMessage(address.MustParseAddr(contractAddr), tlb.MustFromTON("0.01"), body)

	log.Println("sending test dict transaction...")
	tx, _, err := w.SendWaitTransaction(ctx, msg)
	if err != nil {
		panic(err)
	}
	txHash := hex.EncodeToString(tx.Hash)
	// log.Println("transaction confirmed, hash:", txHash)
	return nil, txHash
}

// SetTestDictData 设置TestDict合约数据
func SetTestDictData(testDictAddr string) error {
	// read from config file
	cfg, err := GetParamsCfg()
	if nil != err {
		return err
	}
	if "" == testDictAddr {
		testDictAddr = cfg.CrashGameCfg.ContractAddr
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
	log.Println("start to set dict data...")
	txHash := ""
	if err, txHash = SetDictData(context.Background(), testDictAddr, w); err != nil {
		log.Fatal(err)
	}
	log.Printf(GetScanCfg()+"transaction/%s\n", txHash)
	return nil
}

func Test_Deploy_TestDict(t *testing.T) {
	flag.Parse()
	GetTonAPIIns()
	// 部署合约
	DeployTestDict()
}

func Test_Get_Dict_Data(t *testing.T) {
	flag.Parse()
	GetTonAPIIns()
	GetTestDictData("")
}

func Test_Set_Dict_Data(t *testing.T) {
	flag.Parse()
	GetTonAPIIns()
	SetTestDictData("")
}

func Test_Get_Key_By_ID(t *testing.T) {
	flag.Parse()
	GetTonAPIIns()
	keyId := uint64(0)
	GetKeyByID("", keyId, t)

}
