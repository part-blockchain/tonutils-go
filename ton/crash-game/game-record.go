package crash_game

import (
	"context"
	"fmt"
	"github.com/xssnick/tonutils-go/address"
	"github.com/xssnick/tonutils-go/ton"
	"github.com/xssnick/tonutils-go/tvm/cell"
	"math/big"
)

// GameRecordData game record合约存储数据
type GameRecordData struct {
	RoundIndex       uint64           //  轮数索引(用于合约初始化时，生成合约地址)
	RoundNum         uint64           //  游戏轮数
	Seed             uint64           //  随机数种子
	CrashMultiple    uint64           //  爆炸倍数, 单位:%
	GameState        uint64           //  游戏状态: 0-bet; 1-游戏结束/未启动
	PlayerNums       uint64           //  玩家数量
	CrashGameAddr    *address.Address //  CrashGame合约地址
	JettonMinterAddr *address.Address //  JettonMinter合约地址
	JettonWalletCode *cell.Cell       //  JettonWallet合约代码
	GameWalletCode   *cell.Cell       //  GameWallet合约代码
}

type GameRecordClient struct {
	addr *address.Address
	api  TonApi
}

func NewGameRecordClient(api TonApi, gameWalletAddr *address.Address) *GameRecordClient {
	return &GameRecordClient{
		addr: gameWalletAddr,
		api:  api,
	}
}

// GetGameRecordData 获取GameRecord合约的数据信息
func (c *GameRecordClient) GetGameRecordData(ctx context.Context, showCode bool) (*GameRecordData, error) {
	b, err := c.api.CurrentMasterchainInfo(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get masterchain info: %w", err)
	}
	return c.GetGameRecordDataAtBlock(ctx, b, showCode)
}

// GetGameRecordDataAtBlock 解析合约返回的数据
func (c *GameRecordClient) GetGameRecordDataAtBlock(ctx context.Context, b *ton.BlockIDExt, showCode bool) (*GameRecordData, error) {
	res, err := c.api.WaitForBlock(b.SeqNo).RunGetMethod(ctx, b, c.addr, "get_info")
	if err != nil {
		return nil, fmt.Errorf("failed to run get_info method by GameWallet contract: %w", err)
	}

	index := uint(0)
	data := &GameRecordData{
		RoundIndex:       getValueFromExecutionResult(res, &index, "RoundIndex", false).(*big.Int).Uint64(),     //  游戏轮数
		RoundNum:         getValueFromExecutionResult(res, &index, "RoundNum", false).(*big.Int).Uint64(),       //  游戏轮数
		Seed:             getValueFromExecutionResult(res, &index, "Seed", false).(*big.Int).Uint64(),           //  随机种子
		CrashMultiple:    getValueFromExecutionResult(res, &index, "CrashMultiple", false).(*big.Int).Uint64(),  //  爆炸倍数
		GameState:        getValueFromExecutionResult(res, &index, "GameState", false).(*big.Int).Uint64(),      //  游戏状态
		PlayerNums:       getValueFromExecutionResult(res, &index, "PlayerNums", false).(*big.Int).Uint64(),     //  玩家数量
		CrashGameAddr:    getValueFromExecutionResult(res, &index, "CrashGameAddr", true).(*address.Address),    //  CrashGame合约地址
		JettonMinterAddr: getValueFromExecutionResult(res, &index, "JettonMinterAddr", true).(*address.Address), //  JettonMinter合约地址
	}

	// 显示合约的code
	if showCode {
		data.JettonWalletCode = getValueFromExecutionResult(res, &index, "JettonWalletCode", false).(*cell.Cell) //  JettonWallet合约代码
		data.GameWalletCode = getValueFromExecutionResult(res, &index, "GameWalletCode", false).(*cell.Cell)     //  GameWallet合约代码
	}

	return data, nil
}
