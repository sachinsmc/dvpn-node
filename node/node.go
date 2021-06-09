package node

import (
	"net/http"
	"path"
	"time"

	"github.com/cosmos/cosmos-sdk/client/flags"
	sdk "github.com/cosmos/cosmos-sdk/types"
	hubtypes "github.com/sentinel-official/hub/types"
	nodetypes "github.com/sentinel-official/hub/x/node/types"
	sessiontypes "github.com/sentinel-official/hub/x/session/types"
	"github.com/spf13/viper"

	"github.com/sentinel-official/dvpn-node/context"
	"github.com/sentinel-official/dvpn-node/types"
)

type Node struct {
	*context.Context
}

func NewNode(ctx *context.Context) *Node {
	return &Node{ctx}
}

func (n *Node) Initialize() error {
	n.Logger().Info("Initializing...")

	result, err := n.Client().QueryNode(n.Address())
	if err != nil {
		return err
	}

	if result == nil {
		return n.register()
	}

	return n.updateInfo()
}

func (n *Node) Start() error {
	n.Logger().Info("Starting...")

	go func() {
		if err := n.jobUpdateStatus(); err != nil {
			panic(err)
		}
	}()

	go func() {
		if err := n.jobUpdateSessions(); err != nil {
			panic(err)
		}
	}()

	var (
		certFile = path.Join(viper.GetString(flags.FlagHome), "tls.crt")
		keyFile  = path.Join(viper.GetString(flags.FlagHome), "tls.key")
	)

	return http.ListenAndServeTLS(n.ListenOn(), certFile, keyFile, n.Router())
}

func (n *Node) register() error {
	n.Logger().Info("Registering node...")

	_, err := n.Client().BroadcastTx(
		nodetypes.NewMsgRegisterRequest(
			n.Operator(),
			n.Provider(),
			n.Price(),
			n.RemoteURL(),
		),
	)

	return err
}

func (n *Node) updateInfo() error {
	n.Logger().Info("Updating node info...")

	_, err := n.Client().BroadcastTx(
		nodetypes.NewMsgUpdateRequest(
			n.Address(),
			n.Provider(),
			n.Price(),
			n.RemoteURL(),
		),
	)

	return err
}

func (n *Node) updateStatus() error {
	n.Logger().Info("Updating node status...")

	_, err := n.Client().BroadcastTx(
		nodetypes.NewMsgSetStatusRequest(
			n.Address(),
			hubtypes.StatusActive,
		),
	)

	return err
}

func (n *Node) updateSessions(items ...*types.Session) error {
	n.Logger().Info("Updating sessions...")

	messages := make([]sdk.Msg, 0, len(items))
	for _, item := range items {
		messages = append(messages,
			sessiontypes.NewMsgUpdateRequest(
				n.Address(),
				sessiontypes.Proof{
					Id:        item.ID,
					Duration:  time.Since(item.ConnectedAt),
					Bandwidth: hubtypes.NewBandwidthFromInt64(item.Download, item.Upload),
				},
				nil,
			),
		)
	}

	_, err := n.Client().BroadcastTx(messages...)
	return err
}
