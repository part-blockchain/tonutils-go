package jetton

import (
	"context"
	"crypto/rand"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"github.com/xssnick/tonutils-go/ton/wallet"
	"log"
	"math/big"

	"github.com/xssnick/tonutils-go/address"
	"github.com/xssnick/tonutils-go/tlb"
	"github.com/xssnick/tonutils-go/ton"
	"github.com/xssnick/tonutils-go/tvm/cell"
)

type TonApi interface {
	WaitForBlock(seqno uint32) ton.APIClientWrapped
	CurrentMasterchainInfo(ctx context.Context) (_ *ton.BlockIDExt, err error)
	RunGetMethod(ctx context.Context, blockInfo *ton.BlockIDExt, addr *address.Address, method string, params ...any) (*ton.ExecutionResult, error)
	SubscribeOnTransactions(workerCtx context.Context, addr *address.Address, lastProcessedLT uint64, channel chan<- *tlb.Transaction)
}

var ErrInvalidTransfer = errors.New("transfer is not verified")

type MintPayloadMasterMsg struct {
	Opcode        uint32           `tlb:"## 32"`
	QueryID       uint64           `tlb:"## 64"`
	JettonAmount  tlb.Coins        `tlb:"."`
	FromAddr      *address.Address `tlb:"addr"`
	ResponseAddr  *address.Address `tlb:"addr"`
	ForwardGasFee tlb.Coins        `tlb:"."`
	RestData      *cell.Cell       `tlb:"."`
}

type MintPayload struct {
	_         tlb.Magic            `tlb:"#00000015"` // opcode: 21
	QueryID   uint64               `tlb:"## 64"`
	ToAddress *address.Address     `tlb:"addr"`
	Amount    tlb.Coins            `tlb:"."`
	MasterMsg MintPayloadMasterMsg `tlb:"^"`
}

type TransferNotification struct {
	_              tlb.Magic        `tlb:"#7362d09c"`
	QueryID        uint64           `tlb:"## 64"`
	Amount         tlb.Coins        `tlb:"."`
	Sender         *address.Address `tlb:"addr"`
	ForwardPayload *cell.Cell       `tlb:"either . ^"`
}

type Data struct {
	TotalSupply *big.Int
	Mintable    bool
	AdminAddr   *address.Address
	Content     ContentAny
	WalletCode  *cell.Cell
}

type Client struct {
	addr *address.Address
	api  TonApi
}

func NewJettonMasterClient(api TonApi, masterContractAddr *address.Address) *Client {
	return &Client{
		addr: masterContractAddr,
		api:  api,
	}
}

// GetJettonWallet 获取Jetton Wallet对象
func (c *Client) GetJettonWallet(ctx context.Context, ownerAddr *address.Address) (*WalletClient, error) {
	b, err := c.api.CurrentMasterchainInfo(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get masterchain info: %w", err)
	}
	return c.GetJettonWalletAtBlock(ctx, ownerAddr, b)
}

func (c *Client) GetJettonWalletAtBlock(ctx context.Context, ownerAddr *address.Address, b *ton.BlockIDExt) (*WalletClient, error) {
	res, err := c.api.WaitForBlock(b.SeqNo).RunGetMethod(ctx, b, c.addr, "get_wallet_address",
		cell.BeginCell().MustStoreAddr(ownerAddr).EndCell().BeginParse())
	if err != nil {
		return nil, fmt.Errorf("failed to run get_wallet_address method: %w", err)
	}

	x, err := res.Slice(0)
	if err != nil {
		return nil, err
	}

	addr, err := x.LoadAddr()
	if err != nil {
		return nil, fmt.Errorf("failed to load address from result slice: %w", err)
	}

	return &WalletClient{
		master: c,
		addr:   addr,
	}, nil
}

// GetJettonData 获取Jetton Minter合约的数据信息
func (c *Client) GetJettonData(ctx context.Context) (*Data, error) {
	b, err := c.api.CurrentMasterchainInfo(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get masterchain info: %w", err)
	}
	return c.GetJettonDataAtBlock(ctx, b)
}

func (c *Client) GetJettonDataAtBlock(ctx context.Context, b *ton.BlockIDExt) (*Data, error) {
	res, err := c.api.WaitForBlock(b.SeqNo).RunGetMethod(ctx, b, c.addr, "get_jetton_data")
	if err != nil {
		return nil, fmt.Errorf("failed to run get_jetton_data method: %w", err)
	}

	supply, err := res.Int(0)
	if err != nil {
		return nil, fmt.Errorf("supply get err: %w", err)
	}

	mintable, err := res.Int(1)
	if err != nil {
		return nil, fmt.Errorf("mintable get err: %w", err)
	}

	adminAddr, err := res.Slice(2)
	if err != nil {
		return nil, fmt.Errorf("admin addr get err: %w", err)
	}
	addr, err := adminAddr.LoadAddr()
	if err != nil {
		return nil, fmt.Errorf("failed to load address from adminAddr slice: %w", err)
	}

	contentCell, err := res.Cell(3)
	if err != nil {
		return nil, fmt.Errorf("content cell get err: %w", err)
	}

	// 解析cell, 获取jetton的content信息
	content, err := GetContentFromCell(contentCell)
	if err != nil {
		return nil, fmt.Errorf("failed to load content from contentCell: %w", err)
	}

	walletCode, err := res.Cell(4)
	if err != nil {
		return nil, fmt.Errorf("wallet code get err: %w", err)
	}

	return &Data{
		TotalSupply: supply,
		Mintable:    mintable.Cmp(big.NewInt(0)) != 0,
		AdminAddr:   addr,
		Content:     content,
		WalletCode:  walletCode,
	}, nil
}

// BuildMintTokenPayload 生成铸币Payload
func (c *Client) BuildMintTokenPayload(receiveAddr, gasFee, forwardGasFee, jettonAmount string, jettonDecimals int, payloadMsg *cell.Cell) (*cell.Cell, error) {

	buf := make([]byte, 8)
	if _, err := rand.Read(buf); err != nil {
		return nil, err
	}
	rnd := binary.LittleEndian.Uint64(buf)
	mintPayload := MintPayload{
		QueryID:   rnd,
		ToAddress: address.MustParseAddr(receiveAddr),
		Amount:    tlb.MustFromTON(gasFee),
	}
	if nil == payloadMsg {
		mintPayloadMasterMsg := MintPayloadMasterMsg{
			Opcode:        OpInternalTransfer,
			QueryID:       rnd,
			JettonAmount:  tlb.MustFromDecimal(jettonAmount, jettonDecimals),
			FromAddr:      nil,
			ResponseAddr:  c.addr,
			ForwardGasFee: tlb.MustFromTON(forwardGasFee),
			RestData:      nil,
		}
		mintPayload.MasterMsg = mintPayloadMasterMsg
	}
	body, err := tlb.ToCell(mintPayload)
	if err != nil {
		return nil, err
	}
	// 由外面传入payloadMsg, 则使用外面传入的payloadMsg
	if nil != payloadMsg {
		body = body.ToBuilder().MustStoreRef(payloadMsg).EndCell()
	}

	return body, nil
}

// MintToken 铸币
func (c *Client) MintToken(ctx *context.Context, w *wallet.Wallet, receiveAddr, amount string, payloadMsg *cell.Cell) (error, string) {
	mintData, err := c.BuildMintTokenPayload(receiveAddr, "0.1", "0.05", amount, 6, payloadMsg)
	if err != nil {
		log.Fatalln("build mint token payload failed:", err.Error())
		return err, ""
	}

	// your TON balance must be > 0.05 to send
	msg := wallet.SimpleMessage(c.addr, tlb.MustFromTON("0.15"), mintData)

	log.Println("sending mint token transaction...")
	tx, _, err := w.SendWaitTransaction(*ctx, msg)
	if err != nil {
		panic(err)
	}
	txHash := hex.EncodeToString(tx.Hash)
	log.Println("transaction confirmed, hash:", txHash)
	return nil, txHash
}
