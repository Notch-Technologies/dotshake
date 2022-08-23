// Copyright (c) 2022 Notch Inc & AUTHORS All rights reserved.
// Use of this source code is governed by a BSD 3-Clause License
// license that can be found in the LICENSE file.

package wonderwall

// wonderwall is a magic wall that coordinates communication with
// ether and rcn control planes, etc. // that assist in communication
//

import (
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

	w.dotlog.Logger.Debugf("dotshake connect status => [%s]", ds.Status)
	w.dotlog.Logger.Debugf("dotshake ip => %s/%s", ds.Ip, ds.Cidr)
	return nil
}

func (w *WonderWall) Start() {
	err := w.dialRcnSock()
	if err != nil {
		w.dotlog.Logger.Errorf("failed to dial rcn sock %s", err.Error())
	}
}
