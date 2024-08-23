package jetton

import (
	"context"
	"crypto/rand"
	"encoding/binary"
	"fmt"
	"math/big"

	"github.com/xssnick/tonutils-go/address"
	"github.com/xssnick/tonutils-go/tlb"
	"github.com/xssnick/tonutils-go/ton"
	"github.com/xssnick/tonutils-go/tvm/cell"
)

type JettonWalletData struct {
	Balance          *big.Int
	OwnerAddr        string
	JettonMinterAddr string
	JettonWalletCode *cell.Cell
}

type TransferPayload struct {
	_                   tlb.Magic        `tlb:"#0f8a7ea5"`
	QueryID             uint64           `tlb:"## 64"`
	Amount              tlb.Coins        `tlb:"."`
	Destination         *address.Address `tlb:"addr"`
	ResponseDestination *address.Address `tlb:"addr"`
	CustomPayload       *cell.Cell       `tlb:"maybe ^"`
	ForwardTONAmount    tlb.Coins        `tlb:"."`
	ForwardPayload      *cell.Cell       `tlb:"either . ^"`
}

type BurnPayload struct {
	_                   tlb.Magic        `tlb:"#595f07bc"`
	QueryID             uint64           `tlb:"## 64"`
	Amount              tlb.Coins        `tlb:"."`
	ResponseDestination *address.Address `tlb:"addr"`
	CustomPayload       *cell.Cell       `tlb:"maybe ^"`
}

type WalletClient struct {
	master *Client
	addr   *address.Address
}

func (c *WalletClient) Address() *address.Address {
	return c.addr
}

func (c *WalletClient) GetBalance(ctx context.Context) (*big.Int, error) {
	b, err := c.master.api.CurrentMasterchainInfo(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get masterchain info: %w", err)
	}
	return c.GetBalanceAtBlock(ctx, b)
}

func (c *WalletClient) GetBalanceAtBlock(ctx context.Context, b *ton.BlockIDExt) (*big.Int, error) {
	res, err := c.master.api.WaitForBlock(b.SeqNo).RunGetMethod(ctx, b, c.addr, "get_wallet_data")
	if err != nil {
		if cErr, ok := err.(ton.ContractExecError); ok && cErr.Code == ton.ErrCodeContractNotInitialized {
			return big.NewInt(0), nil
		}
		return nil, fmt.Errorf("failed to run get_wallet_data method: %w", err)
	}

	balance, err := res.Int(0)
	if err != nil {
		return nil, fmt.Errorf("failed to parse balance: %w", err)
	}

	return balance, nil
}

func (c *WalletClient) GetWalletData(ctx context.Context) (*JettonWalletData, error) {
	b, err := c.master.api.CurrentMasterchainInfo(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get masterchain info: %w", err)
	}
	return c.GetWalletDataAtBlock(ctx, b)
}

func (c *WalletClient) GetWalletDataAtBlock(ctx context.Context, b *ton.BlockIDExt) (*JettonWalletData, error) {
	res, err := c.master.api.WaitForBlock(b.SeqNo).RunGetMethod(ctx, b, c.addr, "get_wallet_data")
	if err != nil {
		if cErr, ok := err.(ton.ContractExecError); ok && cErr.Code == ton.ErrCodeContractNotInitialized {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to run get_wallet_data method: %w", err)
	}

	data := JettonWalletData{}
	balance, err := res.Int(0)
	if err != nil {
		return nil, fmt.Errorf("failed to parse balance: %w", err)
	}
	data.Balance = balance

	ownerAddrSlice, err := res.Slice(1)
	if err != nil {
		return nil, fmt.Errorf("failed to parse owner address: %w", err)
	}
	data.OwnerAddr = ownerAddrSlice.MustLoadAddr().String()

	jettonMinterAddrSlice, err := res.Slice(2)
	if err != nil {
		return nil, fmt.Errorf("failed to parse jetton minter address: %w", err)
	}
	data.JettonMinterAddr = jettonMinterAddrSlice.MustLoadAddr().String()

	jettonWalletCode, err := res.Cell(3)
	if err != nil {
		return nil, fmt.Errorf("failed to parse jetton wallet code: %w", err)
	}
	data.JettonWalletCode = jettonWalletCode
	return &data, nil
}

// Deprecated: use BuildTransferPayloadV2
func (c *WalletClient) BuildTransferPayload(to *address.Address, amountCoins, amountForwardTON tlb.Coins, payloadForward *cell.Cell) (*cell.Cell, error) {
	return c.BuildTransferPayloadV2(to, to, amountCoins, amountForwardTON, payloadForward, nil)
}

func (c *WalletClient) BuildTransferPayloadV2(to, responseTo *address.Address, amountCoins, amountForwardTON tlb.Coins, payloadForward, customPayload *cell.Cell) (*cell.Cell, error) {
	if payloadForward == nil {
		payloadForward = cell.BeginCell().EndCell()
	}

	buf := make([]byte, 8)
	if _, err := rand.Read(buf); err != nil {
		return nil, err
	}
	rnd := binary.LittleEndian.Uint64(buf)

	body, err := tlb.ToCell(TransferPayload{
		QueryID:             rnd,
		Amount:              amountCoins,
		Destination:         to,
		ResponseDestination: responseTo,
		CustomPayload:       customPayload,
		ForwardTONAmount:    amountForwardTON,
		ForwardPayload:      payloadForward,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to convert TransferPayload to cell: %w", err)
	}

	return body, nil
}

func (c *WalletClient) BuildBurnPayload(amountCoins tlb.Coins, notifyAddr *address.Address) (*cell.Cell, error) {
	buf := make([]byte, 8)
	if _, err := rand.Read(buf); err != nil {
		return nil, err
	}
	rnd := binary.LittleEndian.Uint64(buf)

	body, err := tlb.ToCell(BurnPayload{
		QueryID:             rnd,
		Amount:              amountCoins,
		ResponseDestination: notifyAddr,
		CustomPayload:       nil,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to convert BurnPayload to cell: %w", err)
	}

	return body, nil
}
