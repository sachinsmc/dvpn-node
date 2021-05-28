package lite

import (
	"fmt"

	"github.com/avast/retry-go"
	"github.com/cosmos/cosmos-sdk/client/tx"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/errors"
)

func (c *Client) BroadcastTx(messages ...sdk.Msg) (res *sdk.TxResponse, err error) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	account, err := c.AccountRetriever().GetAccount(c.ctx, c.FromAddress())
	if err != nil {
		return nil, err
	}

	var (
		name = c.From()
		txf  = c.txf.
			WithAccountNumber(account.GetAccountNumber()).
			WithSequence(account.GetSequence())
	)

	if c.SimulateAndExecute() {
		_, adjusted, err := tx.CalculateGas(c.ctx.QueryWithData, txf, messages...)
		if err != nil {
			return nil, err
		}

		txf = txf.WithGas(adjusted)
	}

	txb, err := tx.BuildUnsignedTx(txf, messages...)
	if err != nil {
		return nil, err
	}

	if err := tx.Sign(txf, name, txb, true); err != nil {
		return nil, err
	}

	txBytes, err := c.TxConfig().TxEncoder()(txb.GetTx())
	if err != nil {
		return nil, err
	}

	err = retry.Do(
		func() error {
			res, err = c.ctx.BroadcastTx(txBytes)

			switch {
			case err != nil:
				return err
			case res.Code == 0:
				return nil
			case res.Code == errors.ErrTxInMempoolCache.ABCICode():
				return nil
			default:
				return fmt.Errorf(res.RawLog)
			}
		},
		retry.Attempts(5),
	)

	return res, err
}