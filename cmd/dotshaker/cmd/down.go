// Copyright (c) 2022 Notch Inc & AUTHORS All rights reserved.
// Use of this source code is governed by a BSD 3-Clause License
// license that can be found in the LICENSE file.

// the down cmd terminates the dotshaker daemon process and closes
// the p2p connection

package cmd

import (
	"context"
	"flag"
	"fmt"
	"time"

	"github.com/Notch-Technologies/dotshake/daemon"
	dd "github.com/Notch-Technologies/dotshake/daemon/dotshaker"
	"github.com/Notch-Technologies/dotshake/dotlog"
	"github.com/Notch-Technologies/dotshake/paths"
	"github.com/Notch-Technologies/dotshake/rcn"
	"github.com/peterbourgon/ff/v2/ffcli"
)

var downArgs struct {
	logFile  string
	logLevel string
	debug    bool
}

var downCmd = &ffcli.Command{
	Name:      "down",
	ShortHelp: "down the dotshaker",
	FlagSet: (func() *flag.FlagSet {
		fs := flag.NewFlagSet("down", flag.ExitOnError)
		fs.StringVar(&downArgs.logFile, "logfile", paths.DefaultDotShakerLogFile(), "set logfile path")
		fs.StringVar(&downArgs.logLevel, "loglevel", dotlog.InfoLevelStr, "set log level")
		fs.BoolVar(&downArgs.debug, "debug", false, "is debug")
		return fs
	})(),
	Exec: execDown,
}

// uninstall dotshaker and delete wireguard interface
//
func execDown(ctx context.Context, args []string) error {
	dotlog, err := dotlog.NewDotLog("dotshaker down", downArgs.logLevel, downArgs.logFile, downArgs.debug)
	if err != nil {
		fmt.Println("failed to initialize logger")
		return nil
	}

	d := daemon.NewDaemon(dd.BinPath, dd.ServiceName, dd.DaemonFilePath, dd.SystemConfig, dotlog)

	_, isInstalled := d.Status()
	if !isInstalled {
		fmt.Println("already terminated")
		return nil
	}

	clientCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	signalClient, serverClient, clientConf, mPubKey := initializeDotShakerConf(
		clientCtx,
		upArgs.clientPath,
		upArgs.debug, upArgs.serverHost, uint(upArgs.serverPort), upArgs.signalHost, uint(upArgs.signalPort), dotlog)

	r := rcn.NewRcn(signalClient, serverClient, clientConf, mPubKey, nil, dotlog)

	err = r.Stop()
	if err != nil {
		fmt.Println("failed to uninstall dotshake")
		return nil
	}

	err = d.Uninstall()
	if err != nil {
		fmt.Println("failed to uninstall dotshake")
		return nil
	}

	return nil
}
