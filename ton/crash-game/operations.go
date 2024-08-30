package crash_game

/*
游戏操作：
1. 创建新的一轮游戏： new round
2. crash游戏： crash
*/

import (
	"context"
	"crypto/rand"
	"encoding/binary"
	"encoding/hex"
	"github.com/xssnick/tonutils-go/tlb"
	"github.com/xssnick/tonutils-go/ton/wallet"
	"github.com/xssnick/tonutils-go/tvm/cell"
	"log"
)

// NewRoundPayload new round payload
type NewRoundPayload struct {
	_       tlb.Magic `tlb:"#6cd87433"` // new round opcode
	QueryID uint64    `tlb:"## 64"`
}

// CrashPayload crash payload
type CrashPayload struct {
	_             tlb.Magic `tlb:"#71bb125e"` // new round opcode
	QueryID       uint64    `tlb:"## 64"`
	RoundNum      uint64    `tlb:"## 32"` // round number, 32 bits
	ForwardGasFee tlb.Coins `tlb:"."`
}

// BuildNewRoundPayload 生成NewRound的Payload
func (c *Client) BuildNewRoundPayload() (*cell.Cell, error) {

	buf := make([]byte, 8)
	if _, err := rand.Read(buf); err != nil {
		return nil, err
	}
	rnd := binary.LittleEndian.Uint64(buf)
	txPayload := NewRoundPayload{
		QueryID: rnd,
	}

	body, err := tlb.ToCell(txPayload)
	if err != nil {
		return nil, err
	}
	return body, nil
}

// BuildCrashPayload 生成Crash的Payload
func (c *Client) BuildCrashPayload(roundNum uint64, forwardGasFee tlb.Coins) (*cell.Cell, error) {

	buf := make([]byte, 8)
	if _, err := rand.Read(buf); err != nil {
		return nil, err
	}
	rnd := binary.LittleEndian.Uint64(buf)
	log.Println("rnd(QueryID):", rnd)
	txPayload := CrashPayload{
		QueryID:       rnd,
		RoundNum:      roundNum,
		ForwardGasFee: forwardGasFee,
	}

	body, err := tlb.ToCell(txPayload)
	if err != nil {
		return nil, err
	}

	return body, nil
}

// NewRound 创建新的一轮游戏
func (c *Client) NewRound(ctx *context.Context, w *wallet.Wallet) (error, string) {
	newRoundData, err := c.BuildNewRoundPayload()
	if err != nil {
		log.Fatalln("build new round payload failed:", err.Error())
		return err, ""
	}

	// your TON balance must be > 0.05 to send
	msg := wallet.SimpleMessage(c.addr, tlb.MustFromTON("0.05"), newRoundData)

	log.Println("sending new round transaction...")
	tx, _, err := w.SendWaitTransaction(*ctx, msg)
	if err != nil {
		panic(err)
	}
	txHash := hex.EncodeToString(tx.Hash)
	// log.Println("transaction confirmed, hash:", txHash)
	return nil, txHash
}

// Crash crash游戏
func (c *Client) Crash(ctx *context.Context, w *wallet.Wallet, roundNum uint64, gasFee, forwardGasFee tlb.Coins) (error, string) {
	crashPayload, err := c.BuildCrashPayload(roundNum, forwardGasFee)
	if err != nil {
		log.Fatalln("build crash payload failed:", err.Error())
		return err, ""
	}
	log.Printf("crashPayload hash:%s\n", hex.EncodeToString(crashPayload.Hash()))
	// your TON balance must be > 0.05 to send
	msg := wallet.SimpleMessage(c.addr, gasFee, crashPayload)

	log.Println("sending crash transaction...")
	tx, _, err := w.SendWaitTransaction(*ctx, msg)
	if err != nil {
		panic(err)
	}
	txHash := hex.EncodeToString(tx.Hash)
	// log.Println("transaction confirmed, hash:", txHash)
	return nil, txHash
}
