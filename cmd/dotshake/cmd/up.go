// Copyright (c) 2022 Notch Inc & AUTHORS All rights reserved.
// Use of this source code is governed by a BSD 3-Clause License
// license that can be found in the LICENSE file.

package cmd

import (
	"context"
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	grpc_client "github.com/Notch-Technologies/dotshake/client/grpc"
	"github.com/Notch-Technologies/dotshake/daemon"
	dd "github.com/Notch-Technologies/dotshake/daemon/dotshaker"
	"github.com/Notch-Technologies/dotshake/dotengine"
	"github.com/Notch-Technologies/dotshake/dotlog"
	"github.com/Notch-Technologies/dotshake/paths"
	"github.com/Notch-Technologies/dotshake/process"
	"github.com/Notch-Technologies/dotshake/types/flagtype"
	"github.com/peterbourgon/ff/v2/ffcli"
)

var upArgs struct {
	clientPath string
	serverHost string
	serverPort int64
	signalHost string
	signalPort int64
	logFile    string
	logLevel   string
	debug      bool
}

var upCmd = &ffcli.Command{
	Name:       "up",
	ShortUsage: "up [flags]",
	ShortHelp:  "up to dotshake, communication client of dotshake",
	FlagSet: (func() *flag.FlagSet {
		fs := flag.NewFlagSet("up", flag.ExitOnError)
		fs.StringVar(&upArgs.clientPath, "path", paths.DefaultClientConfigFile(), "client default config file")
		fs.StringVar(&upArgs.serverHost, "server-host", "https://ctl.dotshake.com", "server host")
		fs.Int64Var(&upArgs.serverPort, "server-port", flagtype.DefaultServerPort, "grpc server host port")
		fs.StringVar(&upArgs.signalHost, "signal-host", "https://signal.dotshake.com", "signal server host")
		fs.Int64Var(&upArgs.signalPort, "signal-port", flagtype.DefaultSignalingServerPort, "signal server port")
		fs.StringVar(&upArgs.logFile, "logfile", paths.DefaultClientLogFile(), "set logfile path")
		fs.StringVar(&upArgs.logLevel, "loglevel", dotlog.InfoLevelStr, "set log level")
		fs.BoolVar(&upArgs.debug, "debug", false, "is debug")
		return fs
	})(),
	Exec: execUp,
}

// after login, check to see if the dotshaker daemon is up.
// if not, prompt the user to start it.
//
func execUp(ctx context.Context, args []string) error {
	err := dotlog.InitDotLog(upArgs.logLevel, upArgs.logFile, upArgs.debug)
	if err != nil {
		log.Fatalf("failed to initialize logger. because %v", err)
	}
	dotlog := dotlog.NewDotLog("dotshake up")

	clientCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	mPubKey, serverClient, clientConf := initializeDotShakeConf(
		clientCtx, dotlog, upArgs.debug, upArgs.clientPath, upArgs.serverHost, uint(upArgs.serverPort),
		upArgs.signalHost, uint(upArgs.signalPort),
	)

	ip, cidr, err := login(ctx, dotlog, clientConf.GetServerHost(), clientConf.WgPrivateKey, mPubKey, upArgs.debug, serverClient)
	if err != nil {
		dotlog.Logger.Warnf("failed to login, %s", err.Error())
	}

	if !isInstallDotshakerDaemon(dotlog) || !isRunningDotShakerProcess(dotlog) {
		dotlog.Logger.Warnf("You need to activate dotshaker. execute this command 'dotshaker up'")
	}

	err = upEngine(ctx, serverClient, dotlog, clientConf.TunName, mPubKey, ip, cidr, clientConf.WgPrivateKey, clientConf.BlackList)
	if err != nil {
		dotlog.Logger.Warnf("failed to start engine. because %v", err)
		return err
	}

	// TODO: (shinta) impl daemon process

	stop := make(chan struct{})
	go func() {
		c := make(chan os.Signal, 1)
		signal.Notify(c,
			os.Interrupt,
			syscall.SIGTERM,
			syscall.SIGINT,
		)
		select {
		case <-c:
			close(stop)
		case <-ctx.Done():
			close(stop)
		}
	}()
	<-stop

	return nil
}

func upEngine(
	ctx context.Context,
	serverClient grpc_client.ServerClientImpl,
	dotlog *dotlog.DotLog,
	tunName string,
	mPubKey string,
	ip string,
	cidr string,
	wgPrivKey string,
	blackList []string,
) error {
	pctx, cancel := context.WithCancel(ctx)

	engine, err := dotengine.NewDotEngine(
		serverClient,
		dotlog,
		tunName,
		mPubKey,
		ip,
		cidr,
		wgPrivKey,
		blackList,
		pctx,
		cancel,
	)
	if err != nil {
		dotlog.Logger.Warnf("failed to connect signal client. because %v", err)
		return err
	}

	// start engine
	err = engine.Start()
	if err != nil {
		dotlog.Logger.Warnf("failed to start dotengine. because %v", err)
		return err
	}

	return nil
}

func isInstallDotshakerDaemon(dotlog *dotlog.DotLog) bool {
	d := daemon.NewDaemon(dd.BinPath, dd.ServiceName, dd.DaemonFilePath, dd.SystemConfig, dotlog)
	_, isInstalled := d.Status()
	return isInstalled
}

func isRunningDotShakerProcess(dotlog *dotlog.DotLog) bool {
	p := process.NewProcess(dotlog)
	return p.GetDotShakerProcess()
}
