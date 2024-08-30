package crash_game

import (
	"context"
	"fmt"
	"github.com/xssnick/tonutils-go/address"
	"github.com/xssnick/tonutils-go/ton"
	"github.com/xssnick/tonutils-go/tvm/cell"
	"math/big"
)

// GameWalletData game wallet合约存储数据
type GameWalletData struct {
	RoundNum         uint64           //  游戏轮数
	BetAmount        uint64           // 下注金额
	Multiple         uint64           // 玩家乘数, 即爆炸倍数, 单位:%
	WalletOwnerAddr  *address.Address //  GameWallet的owner地址
	CrashGameAddr    *address.Address //  CrashGame合约地址
	JettonMinterAddr *address.Address //  JettonMinter合约地址
	JettonWalletCode *cell.Cell       //  JettonWallet合约代码
	GameRecordCode   *cell.Cell       //  GameRecord合约代码
}

type GameWalletClient struct {
	addr *address.Address
	api  TonApi
}

func NewGameWalletClient(api TonApi, gameWalletAddr *address.Address) *GameWalletClient {
	return &GameWalletClient{
		addr: gameWalletAddr,
		api:  api,
	}
}

// GetGameWalletData 获取GameWallet合约的数据信息
func (c *GameWalletClient) GetGameWalletData(ctx context.Context, showCode bool) (*GameWalletData, error) {
	b, err := c.api.CurrentMasterchainInfo(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get masterchain info: %w", err)
	}
	return c.GetGameWalletDataAtBlock(ctx, b, showCode)
}

// GetGameWalletDataAtBlock 解析合约返回的数据
func (c *GameWalletClient) GetGameWalletDataAtBlock(ctx context.Context, b *ton.BlockIDExt, showCode bool) (*GameWalletData, error) {
	res, err := c.api.WaitForBlock(b.SeqNo).RunGetMethod(ctx, b, c.addr, "get_info")
	if err != nil {
		return nil, fmt.Errorf("failed to run get_info method by GameWallet contract: %w", err)
	}

	data := &GameWalletData{
		RoundNum:         getValueFromExecutionResult(res, 0, "RoundNum", false).(*big.Int).Uint64(),       //  游戏轮数
		BetAmount:        getValueFromExecutionResult(res, 1, "BetAmount", false).(*big.Int).Uint64(),      //  下注金额
		Multiple:         getValueFromExecutionResult(res, 2, "Multiple", false).(*big.Int).Uint64(),       //  玩家指定的乘数
		WalletOwnerAddr:  getValueFromExecutionResult(res, 3, "WalletOwnerAddr", true).(*address.Address),  //  游戏钱包的owner地址
		CrashGameAddr:    getValueFromExecutionResult(res, 4, "CrashGameAddr", true).(*address.Address),    //  CrashGame合约地址
		JettonMinterAddr: getValueFromExecutionResult(res, 5, "JettonMinterAddr", true).(*address.Address), //  JettonMinter合约地址
	}

	// 显示合约的code
	if showCode {
		data.JettonWalletCode = getValueFromExecutionResult(res, 6, "JettonWalletCode", false).(*cell.Cell) //  JettonWallet合约代码
		data.GameRecordCode = getValueFromExecutionResult(res, 7, "gameRecordCode", false).(*cell.Cell)     //  GameRecord合约代码
	}

	return data, nil
}
