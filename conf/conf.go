package conf

import (
	"context"

	grpc_client "github.com/Notch-Technologies/dotshake/client/grpc"
	"github.com/Notch-Technologies/dotshake/dotlog"
	"github.com/Notch-Technologies/dotshake/paths"
	"github.com/Notch-Technologies/dotshake/rcn/conn"
	"github.com/Notch-Technologies/dotshake/store"
	"google.golang.org/grpc"
)

type Conf struct {
	SignalClient  grpc_client.SignalClientImpl
	ServerClient  grpc_client.ServerClientImpl
	Spec          *Spec
	MachinePubKey string
	dotlog        *dotlog.DotLog
}

func NewConf(
	clientCtx context.Context,
	path string,
	isDev bool,
	serverHost string, serverPort uint,
	signalHost string, signalPort uint,
	dotlog *dotlog.DotLog,
) (*Conf, error) {
	// configure file store
	//
	cfs, err := store.NewFileStore(paths.DefaultDotshakeClientStateFile(), dotlog)
	if err != nil {
		dotlog.Logger.Warnf("failed to create clietnt state, because %v", err)
		return nil, err
	}

	// configure client store
	//
	cs := store.NewClientStore(cfs, dotlog)
	err = cs.WritePrivateKey()
	if err != nil {
		dotlog.Logger.Warnf("failed to write client state private key, because %v", err)
		return nil, err
	}

	// initialize client config
	//
	spec, err := NewSpec(
		path,
		serverHost, uint(serverPort),
		signalHost, uint(signalPort),
		isDev,
		dotlog,
	)
	if err != nil {
		dotlog.Logger.Warnf("failed to initialize client core, because %v", err)
		return nil, err
	}

	spec = spec.CreateSpec()

	option := grpc_client.NewGrpcDialOption(dotlog, isDev)

	signalClient, err := setupGrpcSignalClient(clientCtx, spec.GetSignalHost(), dotlog, option)
	if err != nil {
		dotlog.Logger.Warnf("failed to initialize grpc signal client, because %v", err)
		return nil, err
	}

	serverClient, err := setupGrpcServerClient(clientCtx, spec.GetServerHost(), dotlog, option)
	if err != nil {
		dotlog.Logger.Warnf("failed to initialize grpc server client, because %v", err)
		return nil, err
	}

	return &Conf{
		SignalClient:  signalClient,
		ServerClient:  serverClient,
		Spec:          spec,
		MachinePubKey: cs.GetPublicKey(),
	}, nil

}

func setupGrpcServerClient(
	clientctx context.Context,
	url string,
	dotlog *dotlog.DotLog,
	option grpc.DialOption,
) (grpc_client.ServerClientImpl, error) {
	sconn, err := grpc.DialContext(
		clientctx,
		url,
		option,
		grpc.WithBlock(),
	)

	serverClient := grpc_client.NewServerClient(sconn, dotlog)
	if err != nil {
		dotlog.Logger.Warnf("failed to connect server client, because %v", err)
		return nil, err
	}

	return serverClient, err
}

func setupGrpcSignalClient(
	clientctx context.Context,
	url string,
	dotlog *dotlog.DotLog,
	option grpc.DialOption,
) (grpc_client.SignalClientImpl, error) {
	gconn, err := grpc.DialContext(
		clientctx,
		url,
		option,
		grpc.WithBlock(),
	)
	if err != nil {
		dotlog.Logger.Warnf("failed to connect signal client, because %v", err)
		return nil, err
	}

	connState := conn.NewConnectedState()

	signalClient := grpc_client.NewSignalClient(gconn, connState, dotlog)

	return signalClient, err
}
