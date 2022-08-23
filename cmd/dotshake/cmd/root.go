// Copyright (c) 2022 Notch Inc & AUTHORS All rights reserved.
// Use of this source code is governed by a BSD 3-Clause License
// license that can be found in the LICENSE file.

package cmd

import (
	"context"
	"flag"
	"strings"
	"time"

	grpc_client "github.com/Notch-Technologies/dotshake/client/grpc"
	"github.com/Notch-Technologies/dotshake/conf"
	"github.com/Notch-Technologies/dotshake/dotlog"
	"github.com/Notch-Technologies/dotshake/paths"
	"github.com/Notch-Technologies/dotshake/store"
	"github.com/peterbourgon/ff/v2/ffcli"
	"google.golang.org/grpc"
	"google.golang.org/grpc/keepalive"
)

// processing required for all commands in common
//
func initializeDotShakeConf(
	clientCtx context.Context,
	dotlog *dotlog.DotLog,
	isDebug bool,
	clientPath string,
	serverHost string, serverPort uint,
	signalHost string, signalPort uint,
) (mPubKey string, serverClient grpc_client.ServerClientImpl, clientConf *conf.ClientConf) {
	// initialize file store
	//
	cfs, err := store.NewFileStore(paths.DefaultDotshakeClientStateFile(), dotlog)
	if err != nil {
		dotlog.Logger.Fatalf("failed to create clietnt state. because %v", err)
	}

	cs := store.NewClientStore(cfs, dotlog)
	err = cs.WritePrivateKey()
	if err != nil {
		dotlog.Logger.Fatalf("failed to write client state private key. because %v", err)
	}
	mPubKey = cs.GetPublicKey()

	// initialize client conf
	//
	clientConf, err = conf.NewClientConf(
		clientPath,
		serverHost, serverPort,
		signalHost, signalPort,
		isDebug,
		dotlog,
	)

	if err != nil {
		dotlog.Logger.Fatalf("failed to initialize client core. because %v", err)
	}

	clientConf = clientConf.CreateClientConf()
	if err != nil {
		dotlog.Logger.Fatalf("can not get client conf, because %v", err)
	}

	option := grpc_client.NewGrpcDialOption(dotlog, isDebug)

	gconn, err := grpc.DialContext(
		clientCtx,
		clientConf.GetServerHost(),
		option,
		grpc.WithBlock(),
		grpc.WithKeepaliveParams(keepalive.ClientParameters{
			Time:                10 * time.Second,
			Timeout:             10 * time.Second,
			PermitWithoutStream: true,
		}))

	serverClient = grpc_client.NewServerClient(gconn, dotlog)
	if err != nil {
		dotlog.Logger.Fatalf("failed to connect server client. because %v", err)
	}

	return mPubKey, serverClient, clientConf
}

func Run(args []string) error {
	if len(args) == 1 && (args[0] == "-V" || args[0] == "--version" || args[0] == "-v") {
		args = []string{"version"}
	}

	fs := flag.NewFlagSet("dotshake", flag.ExitOnError)

	cmd := &ffcli.Command{
		Name:       "dotshake",
		ShortUsage: "dotshake <subcommands> [command flags]",
		ShortHelp:  "Use WireGuard for easy and secure private connections.",
		LongHelp: strings.TrimSpace(`
All flags can use a single or double hyphen.

For help on subcommands, prefix with -help.

Flags and options are subject to change.
`),
		Subcommands: []*ffcli.Command{
			upCmd,
			loginCmd,
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
