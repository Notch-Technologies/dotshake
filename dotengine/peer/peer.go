// Copyright (c) 2022 Notch Inc & AUTHORS All rights reserved.
// Use of this source code is governed by a BSD 3-Clause License
// license that can be found in the LICENSE file.

package peer

import (
	"sync"

	"github.com/Notch-Technologies/dotshake/client/grpc"
	"github.com/Notch-Technologies/dotshake/dotlog"
)

type Peer struct {
	serverClient grpc.ServerClientImpl

	mk string

	mu *sync.Mutex
	ch chan struct{}

	dotlog *dotlog.DotLog
}

func NewPeer(
	serverClient grpc.ServerClientImpl,
	mk string,
	dotlog *dotlog.DotLog,
) *Peer {
	ch := make(chan struct{})

	return &Peer{
		serverClient: serverClient,

		mk: mk,

		mu: &sync.Mutex{},
		ch: ch,

		dotlog: dotlog,
	}
}

func (p *Peer) Up() error {
	err := p.joinHangoutMachines()
	if err != nil {
		return err
	}

	return nil
}

func (p *Peer) joinHangoutMachines() error {
	res, err := p.serverClient.JoinHangoutMachines(p.mk)
	if err != nil {
		p.dotlog.Logger.Errorf("failed to join hangout machines")
		return err
	}

	p.dotlog.Logger.Debugf("successfully joined the hangout machines by [%s]", p.mk)
	p.dotlog.Logger.Debugf("here are the connecting remote machines => %v", res.GetRemotePeers())

	return nil
}
