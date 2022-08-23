// Copyright (c) 2022 Notch Inc & AUTHORS All rights reserved.
// Use of this source code is governed by a BSD 3-Clause License
// license that can be found in the LICENSE file.

package cmd

import (
	"context"
	"fmt"
	"log"

	"github.com/Notch-Technologies/dotshake/daemon"
	dd "github.com/Notch-Technologies/dotshake/daemon/dotshaker"
	"github.com/Notch-Technologies/dotshake/dotlog"
	"github.com/peterbourgon/ff/v2/ffcli"
)

var statusArgs struct {
	logFile  string
	logLevel string
	debug    bool
}

var statusCmd = &ffcli.Command{
	Name:      "status",
	ShortHelp: "status the daemon",
	Subcommands: []*ffcli.Command{
		statusDaemonCmd,
	},
}

var statusDaemonCmd = &ffcli.Command{
	Name:      "daemon",
	ShortHelp: "status the dotshaker daemon",
	Exec:      statusDaemon,
}

func statusDaemon(ctx context.Context, args []string) error {
	err := dotlog.InitDotLog(statusArgs.logLevel, statusArgs.logFile, statusArgs.debug)
	if err != nil {
		log.Fatalf("failed to initialize logger: %v", err)
	}
	dotlog := dotlog.NewDotLog("status")

	d := daemon.NewDaemon(dd.BinPath, dd.ServiceName, dd.DaemonFilePath, dd.SystemConfig, dotlog)
	status, _ := d.Status()
	fmt.Println(status)
	return nil
}
