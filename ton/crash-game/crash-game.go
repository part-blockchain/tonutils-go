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

type JettonWalletInfo struct {
	JettonMinterAddr *address.Address
	JettonWalletAddr *address.Address
	Balance          *big.Int
}

type GameRecordInfo struct {
	ContractAddr *address.Address
	Data         *GameRecordData
}

type CrashGameInfo struct {
	ContractAddr     *address.Address // CrashGame合约地址
	Data             *Data            // CrashGame合约数据
	JettonWalletInfo JettonWalletInfo // JettonWallet合约信息
	GameRecordInfos  []GameRecordInfo // GameRecord合约信息
}

// GameWalletInfo GameWallet合约信息
type GameWalletInfo struct {
	ContractAddr *address.Address // GameWallet合约地址
	// JettonWalletInfo JettonWalletInfo // GameWallet的JettonWallet合约信息, 游戏过程的token不经过GameWallet合约
	Data *GameWalletData // GameWallet合约数据
}

// PlayerInfo 玩家信息
type PlayerInfo struct {
	PlayerAddr       *address.Address // 玩家地址
	JettonWalletInfo JettonWalletInfo // JettonWallet合约信息
	GameWalletInfo   GameWalletInfo   // GameWallet合约信息
}

// GameInfoPeerRound 每轮游戏轮数信息
type GameInfoPeerRound struct {
	RoundIndex    uint64 //  游戏轮数索引
	RoundNum      uint64 //  游戏轮数
	GameState     uint64 //  游戏状态: 0-bet; 1-游戏结束/未启动
	Seed          uint64 //  随机数种子
	CrashMultiple uint64 // Crash乘数, 即爆炸倍数, 单位:%
	PlayerNums    uint64 // 玩家数量
	StartUnixTime uint64 // 游戏开始时间(unix)
	StartTxTime   uint64 // 游戏开始交易时间(unix)
	StartBlkTime  uint64 // 游戏开始区块时间(unix)
}

// Data crash game合约存储数据
type Data struct {
	CellGameInfo *cell.Cell //  存储游戏游戏信息

	CurrentRoundNum   uint64           // 当前轮数
	MaxRoundsParallel uint64           //  最大并行游戏轮数
	MinIntervalTime   uint64           //  一轮游戏从创建完成到crash的最小时间间隔,单位秒
	AdminAddr         *address.Address //  管理员地址
	JettonMinterAddr  *address.Address //  JettonMinter合约地址
	JettonWalletCode  *cell.Cell       //  JettonWallet合约代码
	GameWalletCode    *cell.Cell       //  GameWallet合约代码
	GameRecordCode    *cell.Cell       //  GameRecord合约代码
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
func (c *Client) GetCrashGameData(ctx context.Context, showCode bool, roundNum uint64) (*Data, error) {
	b, err := c.api.CurrentMasterchainInfo(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get masterchain info: %w", err)
	}
	return c.GetCrashGameDataAtBlock(ctx, b, showCode, roundNum)
}

// GetUserGameWalletAddr 获取用户的游戏钱包地址
func (c *Client) GetUserGameWalletAddr(ctx context.Context, ownerAddr *address.Address) (*address.Address, error) {
	b, err := c.api.CurrentMasterchainInfo(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get masterchain info: %w", err)
	}
	res, err := c.api.WaitForBlock(b.SeqNo).RunGetMethod(ctx, b, c.addr, "get_user_game_wallet_addr",
		cell.BeginCell().MustStoreAddr(ownerAddr).EndCell().BeginParse())
	if err != nil {
		return nil, fmt.Errorf("failed to run get_user_game_wallet_addr method by CrashGame contract: %w", err)
	}
	value, err := res.Slice(0)
	if err != nil {
		return nil, err
	}
	return value.LoadAddr()
}

// GetGameRecordAddr 获取游戏记录地址
func (c *Client) GetGameRecordAddr(ctx context.Context, roundNum uint64) (*address.Address, error) {
	b, err := c.api.CurrentMasterchainInfo(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get masterchain info: %w", err)
	}
	res, err := c.api.WaitForBlock(b.SeqNo).RunGetMethod(ctx, b, c.addr, "get_game_record_addr", roundNum)
	if err != nil {
		return nil, fmt.Errorf("failed to run get_game_record_addr method by CrashGame contract: %w", err)
	}
	value, err := res.Slice(0)
	if err != nil {
		return nil, err
	}
	return value.LoadAddr()
}

// GetRoundNum 获取CrashGame合约的RoundNum
func (c *Client) GetRoundNum(ctx context.Context) (uint64, error) {
	b, err := c.api.CurrentMasterchainInfo(ctx)
	if err != nil {
		return 0, fmt.Errorf("failed to get masterchain info: %w", err)
	}
	res, err := c.api.WaitForBlock(b.SeqNo).RunGetMethod(ctx, b, c.addr, "get_public_params")
	if err != nil {
		return 0, fmt.Errorf("failed to run get_public_params method by CrashGame contract: %w", err)
	}
	index := uint(0) // CurrentRoundNum:当前轮数, 用于统计游戏的总轮数
	return getValueFromExecutionResult(res, &index, "CurrentRoundNum", false).(*big.Int).Uint64(), nil
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
		errMsg := fmt.Errorf("%s get err: %w", keyName, err)
		log.Fatal(errMsg)
		return nil
	}
	*index += 1
	return value
}

func (c *Client) GetCrashGameDataAtBlock(ctx context.Context, b *ton.BlockIDExt, showCode bool, roundNum uint64) (*Data, error) {
	// 获取CrashGame合约的公共数据
	res, err := c.api.WaitForBlock(b.SeqNo).RunGetMethod(ctx, b, c.addr, "get_public_params")
	if err != nil {
		return nil, fmt.Errorf("failed to run get_public_params method by CrashGame contract: %w", err)
	}

	index := uint(0)
	data := &Data{
		CurrentRoundNum:   getValueFromExecutionResult(res, &index, "CurrentRoundNum", false).(*big.Int).Uint64(),   //  当前轮数, 用于统计游戏的总轮数
		MaxRoundsParallel: getValueFromExecutionResult(res, &index, "maxRoundsParallel", false).(*big.Int).Uint64(), //  最大并行游戏轮数
		MinIntervalTime:   getValueFromExecutionResult(res, &index, "minIntervalTime", false).(*big.Int).Uint64(),   //  一轮游戏从创建完成到crash的最小时间间隔,单位秒
		AdminAddr:         getValueFromExecutionResult(res, &index, "adminAddr", true).(*address.Address),           //  管理员地址
		JettonMinterAddr:  getValueFromExecutionResult(res, &index, "jettonMinterAddr", true).(*address.Address),    //  JettonMinter合约地址
	}

	// 显示合约的code
	if showCode {
		data.JettonWalletCode = getValueFromExecutionResult(res, &index, "jettonWalletCode", false).(*cell.Cell) //  JettonWallet合约代码
		data.GameWalletCode = getValueFromExecutionResult(res, &index, "gameWalletCode", false).(*cell.Cell)     //  GameWallet合约代码
		data.GameRecordCode = getValueFromExecutionResult(res, &index, "gameRecordCode", false).(*cell.Cell)     //  GameRecord合约代码
	}

	if roundNum > 0 {
		// 获取CrashGame合约的游戏信息
		res, err := c.api.WaitForBlock(b.SeqNo).RunGetMethod(ctx, b, c.addr, "get_game_info", roundNum)
		if err != nil {
			return nil, fmt.Errorf("failed to run get_game_info method by CrashGame contract: %w, roundNum:%d", err, roundNum)
		}
		index = 0
		data.CellGameInfo = getValueFromExecutionResult(res, &index, "CellGameInfos", false).(*cell.Cell) //  JettonWallet合约代码
	}

	return data, nil
}
