package crash_game

import (
	"context"
	"crypto/rand"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"github.com/xssnick/tonutils-go/address"
	"github.com/xssnick/tonutils-go/tlb"
	"github.com/xssnick/tonutils-go/ton"
	"github.com/xssnick/tonutils-go/ton/wallet"
	"github.com/xssnick/tonutils-go/tvm/cell"
	"log"
	"math/big"
)

// SettlementPayload settlement payload
type SettlementPayload struct {
	_             tlb.Magic        `tlb:"#bbc88046"` // settlement opcode
	QueryID       uint64           `tlb:"## 64"`
	RoundNum      uint64           `tlb:"## 32"` // round number, 32 bits
	RoundIndex    uint64           `tlb:"## 32"` // round index, 32 bits
	ForwardGasFee tlb.Coins        `tlb:"."`
	SettleAddr    *address.Address `tlb:"addr"`
}

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

	index := uint(0)
	data := &GameWalletData{
		RoundNum:         getValueFromExecutionResult(res, &index, "RoundNum", false).(*big.Int).Uint64(),       //  游戏轮数
		BetAmount:        getValueFromExecutionResult(res, &index, "BetAmount", false).(*big.Int).Uint64(),      //  下注金额
		Multiple:         getValueFromExecutionResult(res, &index, "Multiple", false).(*big.Int).Uint64(),       //  玩家指定的乘数
		WalletOwnerAddr:  getValueFromExecutionResult(res, &index, "WalletOwnerAddr", true).(*address.Address),  //  游戏钱包的owner地址
		CrashGameAddr:    getValueFromExecutionResult(res, &index, "CrashGameAddr", true).(*address.Address),    //  CrashGame合约地址
		JettonMinterAddr: getValueFromExecutionResult(res, &index, "JettonMinterAddr", true).(*address.Address), //  JettonMinter合约地址
	}

	// 显示合约的code
	if showCode {
		data.JettonWalletCode = getValueFromExecutionResult(res, &index, "JettonWalletCode", false).(*cell.Cell) //  JettonWallet合约代码
		data.GameRecordCode = getValueFromExecutionResult(res, &index, "gameRecordCode", false).(*cell.Cell)     //  GameRecord合约代码
	}

	return data, nil
}

func getRoundIndexByNum(roundNum, maxRoundsParallel uint64) uint64 {
	return (roundNum - 1) % maxRoundsParallel
}

// BuildSettlementPayload 生成settlement的Payload
func (c *GameWalletClient) BuildSettlementPayload(roundNum, maxRoundsParallel uint64, forwardGasFee tlb.Coins, settleAddr *address.Address) (*cell.Cell, error) {

	buf := make([]byte, 8)
	if _, err := rand.Read(buf); err != nil {
		return nil, err
	}
	rnd := binary.LittleEndian.Uint64(buf)
	log.Println("rnd(QueryID):", rnd)
	txPayload := SettlementPayload{
		QueryID:       rnd,
		RoundNum:      roundNum,
		RoundIndex:    getRoundIndexByNum(roundNum, maxRoundsParallel),
		ForwardGasFee: forwardGasFee,
		SettleAddr:    settleAddr,
	}

	body, err := tlb.ToCell(txPayload)
	if err != nil {
		return nil, err
	}

	return body, nil
}

// Settlement 结算游戏
func (c *GameWalletClient) Settlement(ctx *context.Context, w *wallet.Wallet, roundNum, maxRoundsParallel uint64, gasFee, forwardGasFee tlb.Coins, settleAddr *address.Address) (error, string) {
	settlePayload, err := c.BuildSettlementPayload(roundNum, maxRoundsParallel, forwardGasFee, settleAddr)
	if err != nil {
		log.Fatalln("build settlement payload failed:", err.Error())
		return err, ""
	}
	log.Printf("settlePayload hash:%s\n", hex.EncodeToString(settlePayload.Hash()))
	// your TON balance must be > 0.05 to send
	msg := wallet.SimpleMessage(c.addr, gasFee, settlePayload)

	log.Println("sending settlement transaction...")
	tx, _, err := w.SendWaitTransaction(*ctx, msg)
	if err != nil {
		panic(err)
	}
	txHash := hex.EncodeToString(tx.Hash)
	// log.Println("transaction confirmed, hash:", txHash)
	return nil, txHash
}
