package crash_game

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

type NewRoundPayload struct {
	_       tlb.Magic `tlb:"#6cd87433"` // opcode
	QueryID uint64    `tlb:"## 64"`
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
	log.Println("transaction confirmed, hash:", txHash)
	return nil, txHash
}
