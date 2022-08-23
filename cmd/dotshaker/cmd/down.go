// Copyright (c) 2022 Notch Inc & AUTHORS All rights reserved.
// Use of this source code is governed by a BSD 3-Clause License
// license that can be found in the LICENSE file.

package cmd

import (
	"context"
	"flag"
	"log"

	"github.com/Notch-Technologies/dotshake/daemon"
	dd "github.com/Notch-Technologies/dotshake/daemon/dotshaker"
	"github.com/Notch-Technologies/dotshake/dotlog"
	"github.com/Notch-Technologies/dotshake/paths"
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
		fs.StringVar(&downArgs.logLevel, "loglevel", dotlog.DebugLevelStr, "set log level")
		fs.BoolVar(&downArgs.debug, "debug", false, "is debug")
		return fs
	})(),
	Exec: execDown,
}

// uninstall dotshaker and delete wireguard interface
//
func execDown(ctx context.Context, args []string) error {
	err := dotlog.InitDotLog(downArgs.logLevel, downArgs.logFile, downArgs.debug)
	if err != nil {
		log.Fatalf("failed to initialize logger. because %v", err)
	}
	dotlog := dotlog.NewDotLog("dotshaker down")

	d := daemon.NewDaemon(dd.BinPath, dd.ServiceName, dd.DaemonFilePath, dd.SystemConfig, dotlog)

	_, isInstalled := d.Status()
	if !isInstalled {
		dotlog.Logger.Debugf("already down")
		return nil
	}

	err = d.Uninstall()
	if err != nil {
		dotlog.Logger.Errorf("failed to uninstall dotshaker, %s", err.Error())
	}

	dotlog.Logger.Debugf("completed down dotshaker")

	return nil
}
