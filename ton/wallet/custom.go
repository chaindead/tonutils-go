package wallet

import (
	"context"
	"crypto/ed25519"
	"github.com/chaindead/tonutils-go/tlb"
	"github.com/chaindead/tonutils-go/tvm/cell"
)

type ConfigCustom interface {
	GetStateInit(pubKey ed25519.PublicKey, subWallet uint32) (*tlb.StateInit, error)
	GetSpec(w *Wallet) MessageBuilder
}

type MessageBuilder interface {
	BuildMessage(ctx context.Context, messages []*Message) (*cell.Cell, error)
}
