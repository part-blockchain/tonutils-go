package crash_game

import (
	"context"
	"errors"
	"fmt"
	"github.com/xssnick/tonutils-go/address"
	"github.com/xssnick/tonutils-go/tlb"
	"github.com/xssnick/tonutils-go/ton"
	"github.com/xssnick/tonutils-go/tvm/cell"
	"log"
	"math/big"
)

type TonApi interface {
	WaitForBlock(seqno uint32) ton.APIClientWrapped
	CurrentMasterchainInfo(ctx context.Context) (_ *ton.BlockIDExt, err error)
	RunGetMethod(ctx context.Context, blockInfo *ton.BlockIDExt, addr *address.Address, method string, params ...any) (*ton.ExecutionResult, error)
	SubscribeOnTransactions(workerCtx context.Context, addr *address.Address, lastProcessedLT uint64, channel chan<- *tlb.Transaction)
}

// Data crash game合约存储数据
type Data struct {
	RoundNum         uint64           //  游戏轮数
	GameState        uint64           //  游戏状态: 0-bet; 1-游戏结束/未启动
	Seed             uint64           //  随机数种子
	CrashMultiple    uint64           // Crash乘数, 即爆炸倍数, 单位:%
	PlayerNums       uint64           // 玩家数量
	StartUnixTime    uint64           // 游戏开始时间(unix)
	StartTxTime      uint64           // 游戏开始交易时间(unix)
	StartBlkTime     uint64           // 游戏开始区块时间(unix)
	MinIntervalTime  uint64           //  一轮游戏从创建完成到crash的最小时间间隔,单位秒
	AdminAddr        *address.Address //  管理员地址
	JettonMinterAddr *address.Address //  JettonMinter合约地址
	JettonWalletCode *cell.Cell       //  JettonWallet合约代码
	GameWalletCode   *cell.Cell       //  GameWallet合约代码
	GameRecordCode   *cell.Cell       //  GameRecord合约代码
}

type Client struct {
	addr *address.Address
	api  TonApi
}

func NewCrashGameClient(api TonApi, crashGameContractAddr *address.Address) *Client {
	return &Client{
		addr: crashGameContractAddr,
		api:  api,
	}
}

// GetCrashGameData 获取CrashGame合约的数据信息
func (c *Client) GetCrashGameData(ctx context.Context, showCode bool) (*Data, error) {
	b, err := c.api.CurrentMasterchainInfo(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get masterchain info: %w", err)
	}
	return c.GetCrashGameDataAtBlock(ctx, b, showCode)
}

// 从执行结果中获取对应的值
func getValueFromExecutionResult(res *ton.ExecutionResult, index uint, keyName string, isAddrType bool) (value interface{}) {
	if res == nil || res.Length() <= index {
		errMsg := fmt.Errorf("%s get err: invalid result or index:%d", keyName, index)
		log.Fatal(errMsg)
		return nil
	}
	typeName := res.Type(index)
	if typeName == "null" {
		// 无效的null类型
		errMsg := fmt.Errorf("%s get err: invalid null type:%d", keyName, index)
		log.Fatal(errMsg)
		return nil
	}
	err := errors.New("")
	switch typeName {
	case "*big.Int":
		value, err = res.Int(index)
	case "*cell.Slice":
		value, err = res.Slice(index)
		if isAddrType && value != nil && err == nil {
			value, err = value.(*cell.Slice).LoadAddr()
		}
	case "*cell.Cell":
		value, err = res.Cell(index)
	}

	if err != nil {
		errMsg := fmt.Errorf("%s get err: %w", keyName, err)
		log.Fatal(errMsg)
		return nil
	}
	return value
}

func (c *Client) GetCrashGameDataAtBlock(ctx context.Context, b *ton.BlockIDExt, showCode bool) (*Data, error) {
	res, err := c.api.WaitForBlock(b.SeqNo).RunGetMethod(ctx, b, c.addr, "get_info")
	if err != nil {
		return nil, fmt.Errorf("failed to run get_info method by CrashGame contract: %w", err)
	}

	data := &Data{
		RoundNum:         getValueFromExecutionResult(res, 0, "roundNum", false).(*big.Int).Uint64(),        //  游戏轮数
		GameState:        getValueFromExecutionResult(res, 1, "gameState", false).(*big.Int).Uint64(),       //  游戏状态: 0-bet; 1-游戏结束/未启动
		Seed:             getValueFromExecutionResult(res, 2, "seed", false).(*big.Int).Uint64(),            //  随机数种子
		CrashMultiple:    getValueFromExecutionResult(res, 3, "crashMultiple", false).(*big.Int).Uint64(),   // Crash乘数, 即爆炸倍数, 单位:%
		PlayerNums:       getValueFromExecutionResult(res, 4, "playerNums", false).(*big.Int).Uint64(),      // 玩家数量
		StartUnixTime:    getValueFromExecutionResult(res, 5, "startUnixTime", false).(*big.Int).Uint64(),   // 游戏开始时间(unix)
		StartTxTime:      getValueFromExecutionResult(res, 6, "startTxTime", false).(*big.Int).Uint64(),     // 游戏开始交易时间(unix)
		StartBlkTime:     getValueFromExecutionResult(res, 7, "startBlkTime", false).(*big.Int).Uint64(),    // 游戏开始区块时间(unix)
		MinIntervalTime:  getValueFromExecutionResult(res, 8, "minIntervalTime", false).(*big.Int).Uint64(), //  一轮游戏从创建完成到crash的最小时间间隔,单位秒
		AdminAddr:        getValueFromExecutionResult(res, 9, "adminAddr", true).(*address.Address),         //  管理员地址
		JettonMinterAddr: getValueFromExecutionResult(res, 10, "jettonMinterAddr", true).(*address.Address), //  JettonMinter合约地址
	}

	// 显示合约的code
	if showCode {
		data.JettonWalletCode = getValueFromExecutionResult(res, 11, "jettonWalletCode", false).(*cell.Cell) //  JettonWallet合约代码
		data.GameWalletCode = getValueFromExecutionResult(res, 12, "gameWalletCode", false).(*cell.Cell)     //  GameWallet合约代码
		data.GameRecordCode = getValueFromExecutionResult(res, 13, "gameRecordCode", false).(*cell.Cell)     //  GameRecord合约代码
	}

	return data, nil
}
