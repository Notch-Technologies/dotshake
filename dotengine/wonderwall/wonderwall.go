// Copyright (c) 2022 Notch Inc & AUTHORS All rights reserved.
// Use of this source code is governed by a BSD 3-Clause License
// license that can be found in the LICENSE file.

package wonderwall

// wonderwall is a magic wall that coordinates communication with
// runs on dotshaker daemon, exchanges status with RCN control plane with server
// so in most cases you can get information about the dotshaker daemon through the server from here.

import (
	"fmt"
	"sync"

	"github.com/Notch-Technologies/dotshake/dotlog"
	"github.com/Notch-Technologies/dotshake/rcn/rcnsock"
)

type WonderWall struct {
	sock *rcnsock.RcnSock

	mu *sync.Mutex

	dotlog *dotlog.DotLog
}

func NewWonderWall(
	sock *rcnsock.RcnSock,
	dotlog *dotlog.DotLog,
) *WonderWall {
	return &WonderWall{
		sock:   sock,
		mu:     &sync.Mutex{},
		dotlog: dotlog,
	}
}

func (w *WonderWall) dialRcnSock() error {
	ds, err := w.sock.DialDotshakeStatus()
	if err != nil {
		return err
	}

	fmt.Printf("dotshake connect to server status => [%s]\n", ds.Status)
	fmt.Printf("dotshake ip => [%s/%s]\n", ds.Ip, ds.Cidr)

	return nil
}

func (w *WonderWall) Start() {
	err := w.dialRcnSock()
	if err != nil {
		w.dotlog.Logger.Errorf("failed to dial rcn sock %s", err.Error())
	}
}
