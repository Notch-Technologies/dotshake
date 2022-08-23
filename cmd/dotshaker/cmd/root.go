// Copyright (c) 2022 Notch Inc & AUTHORS All rights reserved.
// Use of this source code is governed by a BSD 3-Clause License
// license that can be found in the LICENSE file.

package cmd

// dotshaker commands is an always running daemon process that provides the necessary
// functionality for the dotshake command
// it is a behind-the-scenes process that assists
// udp hole punching and the relay of packets to be implemented in the future.

import (
	"context"
	"flag"
	"strings"

	grpc_client "github.com/Notch-Technologies/dotshake/client/grpc"
	"github.com/Notch-Technologies/dotshake/conf"
	"github.com/Notch-Technologies/dotshake/dotlog"
	"github.com/Notch-Technologies/dotshake/paths"
	"github.com/Notch-Technologies/dotshake/rcn/conn"
	"github.com/Notch-Technologies/dotshake/store"
	"github.com/peterbourgon/ff/v2/ffcli"
	"google.golang.org/grpc"
)

func initializeDotShakerConf(
	clientCtx context.Context,
	path string,
	isDev bool,
	serverHost string, serverPort uint,
	signalHost string, signalPort uint,
	dotlog *dotlog.DotLog,
) (signalClient grpc_client.SignalClientImpl, serverClient grpc_client.ServerClientImpl, clientConf *conf.ClientConf, mPubKey string) {
	// configure file store
	//
	cfs, err := store.NewFileStore(paths.DefaultDotshakeClientStateFile(), dotlog)
	if err != nil {
		dotlog.Logger.Fatalf("failed to create clietnt state. because %v", err)
	}

	// configure client store
	//
	cs := store.NewClientStore(cfs, dotlog)
	err = cs.WritePrivateKey()
	if err != nil {
		dotlog.Logger.Fatalf("failed to write client state private key. because %v", err)
	}
	mPubKey = cs.GetPublicKey()

	// initialize client conf
	//
	clientConf, err = conf.NewClientConf(
		path,
		serverHost, uint(serverPort),
		signalHost, uint(signalPort),
		isDev,
		dotlog,
	)
	if err != nil {
		dotlog.Logger.Fatalf("failed to initialize client core. because %v", err)
	}

	clientConf = clientConf.CreateClientConf()

	option := grpc_client.NewGrpcDialOption(dotlog, isDev)

	signalClient, err = setupGrpcSignalClient(clientCtx, clientConf.GetSignalHost(), dotlog, option)
	if err != nil {
		dotlog.Logger.Fatalf("failed to initialize grpc signal client. because %v", err)
	}

	serverClient, err = setupGrpcServerClient(clientCtx, clientConf.GetServerHost(), dotlog, option)
	if err != nil {
		dotlog.Logger.Fatalf("failed to initialize grpc server client. because %v", err)
	}

	return signalClient, serverClient, clientConf, mPubKey
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
		dotlog.Logger.Fatalf("failed to connect server client. because %v", err)
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
		dotlog.Logger.Fatalf("failed to connect signal client. because %v", err)
	}

	connState := conn.NewConnectedState()

	signalClient := grpc_client.NewSignalClient(gconn, connState, dotlog)

	return signalClient, err
}

func Run(args []string) error {
	if len(args) == 1 && (args[0] == "-V" || args[0] == "--version" || args[0] == "-v") {
		args = []string{"version"}
	}

	fs := flag.NewFlagSet("dotshaker", flag.ExitOnError)
	cmd := &ffcli.Command{
		Name:       "dotshaker",
		ShortUsage: "dotshaker <subcommands> [command flags]",
		ShortHelp:  "daemon that provides various functions needed to use dotshaker with dotshake.",
		LongHelp: strings.TrimSpace(`
All flags can use a single or double hyphen.

For help on subcommands, prefix with -help.

Flags and options are subject to change.
`),
		Subcommands: []*ffcli.Command{
			daemonCmd,
			upCmd,
			downCmd,
			statusCmd,
			versionCmd,
		},
		FlagSet: fs,
		Exec:    func(context.Context, []string) error { return flag.ErrHelp },
	}

	if err := cmd.Parse(args); err != nil {
		return err
	}

	if err := cmd.Run(context.Background()); err != nil {
		if err == flag.ErrHelp {
			return nil
		}
		return err
	}

	return nil
}
