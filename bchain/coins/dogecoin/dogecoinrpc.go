package dogecoin

import (
	"encoding/json"
	"math/big"

	"github.com/golang/glog"
	"github.com/trezor/blockbook/bchain"
	"github.com/trezor/blockbook/bchain/coins/btc"
)

// DogecoinRPC is an interface to JSON-RPC dogecoind service.
type DogecoinRPC struct {
	*btc.BitcoinRPC
}

// NewDogecoinRPC returns new DogecoinRPC instance.
func NewDogecoinRPC(config json.RawMessage, pushHandler func(bchain.NotificationType)) (bchain.BlockChain, error) {
	b, err := btc.NewBitcoinRPC(config, pushHandler)
	if err != nil {
		return nil, err
	}

	s := &DogecoinRPC{
		b.(*btc.BitcoinRPC),
	}
	s.RPCMarshaler = btc.JSONMarshalerV1{}
	s.ChainConfig.SupportsEstimateFee = false
	s.MinFeePerKB = 100000 // 0.001 DOGE/kB

	return s, nil
}

// EstimateSmartFee returns fee estimation.
// Dogecoin's estimatesmartfee only accepts nblocks, not estimate_mode.
func (b *DogecoinRPC) EstimateSmartFee(blocks int, conservative bool) (big.Int, error) {
	glog.V(1).Info("rpc: estimatesmartfee ", blocks)

	res := btc.ResEstimateSmartFee{}
	req := struct {
		Method string `json:"method"`
		Params []int  `json:"params"`
	}{
		Method: "estimatesmartfee",
		Params: []int{blocks},
	}

	err := b.Call(&req, &res)

	var r big.Int
	if err != nil {
		return r, err
	}
	if res.Error != nil {
		return r, res.Error
	}
	r, err = b.Parser.AmountToBigInt(res.Result.Feerate)
	if err != nil {
		return r, err
	}
	return b.ApplyMinFee(r), nil
}

// Initialize initializes DogecoinRPC instance.
func (b *DogecoinRPC) Initialize() error {
	ci, err := b.GetChainInfo()
	if err != nil {
		return err
	}
	chainName := ci.Chain

	glog.Info("Chain name ", chainName)
	params := GetChainParams(chainName)

	// always create parser
	b.Parser = NewDogecoinParser(params, b.ChainConfig)

	// parameters for getInfo request
	if params.Net == MainnetMagic {
		b.Testnet = false
		b.Network = "livenet"
	} else {
		b.Testnet = true
		b.Network = "testnet"
	}

	glog.Info("rpc: block chain ", params.Name)

	return nil
}

// GetBlock returns block with given hash.
func (b *DogecoinRPC) GetBlock(hash string, height uint32) (*bchain.Block, error) {
	var err error
	if hash == "" {
		hash, err = b.GetBlockHash(height)
		if err != nil {
			return nil, err
		}
	}
	if !b.ParseBlocks {
		return b.GetBlockFull(hash)
	}
	return b.GetBlockWithoutHeader(hash, height)
}
