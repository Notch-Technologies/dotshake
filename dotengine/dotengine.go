// Copyright (c) 2022 Notch Inc & AUTHORS All rights reserved.
// Use of this source code is governed by a BSD 3-Clause License
// license that can be found in the LICENSE file.

package dotengine

import (
	"context"
	"errors"
	"sync"

	"github.com/Notch-Technologies/dotshake/client/grpc"
	"github.com/Notch-Technologies/dotshake/dotengine/peer"
	"github.com/Notch-Technologies/dotshake/dotengine/wonderwall"
	"github.com/Notch-Technologies/dotshake/dotlog"
	"github.com/Notch-Technologies/dotshake/rcn/rcnsock"
	"github.com/Notch-Technologies/dotshake/wireguard"
	"golang.zx2c4.com/wireguard/wgctrl/wgtypes"
)

type DotEngine struct {
	dotlog *dotlog.DotLog

	mk        string
	tunName   string
	ip        string
	cidr      string
	wgPrivKey string
	wgPort    int
	blackList []string

	peer *peer.Peer

	sock *rcnsock.RcnSock

	ctx    context.Context
	cancel context.CancelFunc

	mu *sync.Mutex

	rootch chan struct{}
}

func NewDotEngine(
	serverClient grpc.ServerClientImpl,
	dotlog *dotlog.DotLog,
	tunName string,
	mk string,
	ip string,
	cidr string,
	wgPrivKey string,
	blackList []string,
	ctx context.Context,
	cancel context.CancelFunc,
) (*DotEngine, error) {
	_, err := wgtypes.ParseKey(wgPrivKey)
	if err != nil {
		return nil, err
	}

	ch := make(chan struct{})
	mu := &sync.Mutex{}

	sock := rcnsock.NewRcnSock(dotlog, ch)

	return &DotEngine{
		dotlog: dotlog,

		mk:        mk,
		tunName:   tunName,
		ip:        ip,
		cidr:      cidr,
		wgPrivKey: wgPrivKey,
		wgPort:    wireguard.WgPort,
		blackList: blackList,

		peer: peer.NewPeer(serverClient, mk, dotlog),

		sock: sock,

		ctx:    ctx,
		cancel: cancel,

		mu: mu,

		rootch: ch,
	}, nil
}

func (d *DotEngine) startWonderWall() {
	ww := wonderwall.NewWonderWall(d.sock, d.dotlog)
	ww.Start()
}

func (d *DotEngine) Start() error {
	d.mu.Lock()
	defer d.mu.Unlock()

	d.startWonderWall()
	// d.peer.Up()

	go func() {
		// do somethings
		// system resouce check?
	}()
	<-d.rootch

	return errors.New("stop the dotengine")
}
