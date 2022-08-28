// Copyright (c) 2022 Notch Inc & AUTHORS All rights reserved.
// Use of this source code is governed by a BSD 3-Clause License
// license that can be found in the LICENSE file.

package cmd

import (
	"context"
	"flag"
	"fmt"
	"log"
	"time"

	grpc_client "github.com/Notch-Technologies/dotshake/client/grpc"
	"github.com/Notch-Technologies/dotshake/dotlog"
	"github.com/Notch-Technologies/dotshake/paths"
	"github.com/Notch-Technologies/dotshake/types/flagtype"
	"github.com/peterbourgon/ff/v2/ffcli"
	"golang.zx2c4.com/wireguard/wgctrl/wgtypes"
)

var loginArgs struct {
	clientPath string
	serverHost string
	serverPort int64
	signalHost string
	signalPort int64
	logFile    string
	logLevel   string
	debug      bool
}

var loginCmd = &ffcli.Command{
	Name:       "login",
	ShortUsage: "login [flags]",
	ShortHelp:  "login to dotshake, start the management server and then run it",
	FlagSet: (func() *flag.FlagSet {
		fs := flag.NewFlagSet("login", flag.ExitOnError)
		fs.StringVar(&loginArgs.clientPath, "path", paths.DefaultClientConfigFile(), "client default config file")
		fs.StringVar(&loginArgs.serverHost, "server-host", "", "grpc server host url")
		fs.Int64Var(&loginArgs.serverPort, "server-port", flagtype.DefaultServerPort, "grpc server host port")
		fs.StringVar(&loginArgs.signalHost, "signal-host", "", "grpc server host url")
		fs.Int64Var(&loginArgs.signalPort, "signal-port", flagtype.DefaultSignalingServerPort, "grpc server host port")
		fs.StringVar(&loginArgs.logFile, "logfile", paths.DefaultClientLogFile(), "set logfile path")
		fs.StringVar(&loginArgs.logLevel, "loglevel", dotlog.DebugLevelStr, "set log level")
		fs.BoolVar(&loginArgs.debug, "debug", false, "is debug")
		return fs
	})(),
	Exec: execLogin,
}

func execLogin(ctx context.Context, args []string) error {
	err := dotlog.InitDotLog(loginArgs.logLevel, loginArgs.logFile, loginArgs.debug)
	if err != nil {
		log.Fatalf("failed to initialize logger. because %v", err)
	}

	dotlog := dotlog.NewDotLog("dotshake login")

	clientCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	mPubKey, serverClient, clientConf := initializeDotShakeConf(
		clientCtx, dotlog, loginArgs.debug, loginArgs.clientPath,
		loginArgs.serverHost, uint(loginArgs.serverPort),
		loginArgs.signalHost, uint(loginArgs.signalPort),
	)

	ip, cidr, err := login(ctx, dotlog, clientConf.GetServerHost(), clientConf.WgPrivateKey, mPubKey, loginArgs.debug, serverClient)
	if err != nil {
		dotlog.Logger.Fatalf("failed to login, %s", err.Error())
	}

	fmt.Printf("Your dotshake ip => [%s/%s]\n", ip, cidr)
	fmt.Printf("Successful login\n")

	return nil
}

func login(
	ctx context.Context,
	dotlog *dotlog.DotLog,
	serverHost string,
	wgPrivKey, mkPubKey string,
	isDev bool,
	serverClient grpc_client.ServerClientImpl,
) (ip string, cidr string, err error) {
	wgPrivateKey, err := wgtypes.ParseKey(wgPrivKey)
	if err != nil {
		dotlog.Logger.Fatalf("failed to parse wg private key. because %v", err)
	}

	res, err := serverClient.GetMachine(mkPubKey, wgPrivateKey.PublicKey().String())
	if err != nil {
		return ip, cidr, err
	}

	// TODO: (shinta) use the open command to make URL pages open by themselves
	if !res.IsRegistered {
		fmt.Printf("please log in via this link => %s\n", res.LoginUrl)
		msg, err := serverClient.ConnectStreamPeerLoginSession(mkPubKey)
		if err != nil {
			return ip, cidr, err
		}

		ip = msg.Ip
		cidr = msg.Cidr

		return ip, cidr, err
	}

	ip = res.Ip
	cidr = res.Cidr

	return ip, cidr, err
}
