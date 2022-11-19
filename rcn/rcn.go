// Copyright (c) 2022 Notch Inc & AUTHORS All rights reserved.
// Use of this source code is governed by a BSD 3-Clause License
// license that can be found in the LICENSE file.

package rcn

// rcn package is realtime communication nucleus
// provides communication status and P2P communication aids
// you must be logged in to use it
//

import (
	"sync"

	"github.com/Notch-Technologies/dotshake/client/grpc"
	"github.com/Notch-Technologies/dotshake/conf"
	"github.com/Notch-Technologies/dotshake/dotlog"
	"github.com/Notch-Technologies/dotshake/iface"
	"github.com/Notch-Technologies/dotshake/rcn/controlplane"
	"github.com/Notch-Technologies/dotshake/rcn/rcnsock"
	"golang.zx2c4.com/wireguard/wgctrl/wgtypes"
)

type Rcn struct {
	cp           *controlplane.ControlPlane
	serverClient grpc.ServerClientImpl
	conf         *conf.Conf
	iface        *iface.Iface
	mk           string
	mu           *sync.Mutex
	dotlog       *dotlog.DotLog
}

func NewRcn(
	conf *conf.Conf,
	mk string,
	ch chan struct{},
	dotlog *dotlog.DotLog,
) *Rcn {
	cp := controlplane.NewControlPlane(
		conf.SignalClient,
		conf.ServerClient,
		rcnsock.NewRcnSock(dotlog, ch),
		mk,
		conf,
		ch,
		dotlog,
	)

	return &Rcn{
		cp:           cp,
		serverClient: conf.ServerClient,
		conf:         conf,
		mk:           mk,
		mu:           &sync.Mutex{},
		dotlog:       dotlog,
	}
}

func (r *Rcn) Start() {
	err := r.createIface()
	if err != nil {
		r.dotlog.Logger.Errorf("failed to create iface, %s", err.Error())
	}

	err = r.cp.ConfigureStunTurnConf()
	if err != nil {
		r.dotlog.Logger.Errorf("failed to set up puncher, %s", err.Error())
	}

	go r.cp.ConnectSignalServer()

	go r.cp.WaitForRemoteConn()

	go r.cp.SyncRemoteMachine()

	r.dotlog.Logger.Debugf("started rcn")
}

func (r *Rcn) createIface() error {
	wgPrivateKey, err := wgtypes.ParseKey(r.conf.Spec.WgPrivateKey)
	if err != nil {
		r.dotlog.Logger.Warnf("failed to parse wg private key, because %v", err)
		return err
	}

	res, err := r.serverClient.Login(r.mk, wgPrivateKey.PublicKey().String())
	if err != nil {
		return err
	}

	if !res.IsRegistered {
		r.dotlog.Logger.Warnf("please login with `dotshake login` and try again")
	}

	r.iface = iface.NewIface(r.conf.Spec.TunName, r.conf.Spec.WgPrivateKey, res.Ip, res.Cidr, r.dotlog)
	return iface.CreateIface(r.iface, r.dotlog)
}

func (r *Rcn) Stop() error {
	err := r.cp.Close()
	if err != nil {
		r.dotlog.Logger.Errorf("failed to close control plane, because %s", err.Error())
		return err
	}

	err = iface.RemoveIface(r.iface.Tun, r.dotlog)
	if err != nil {
		r.dotlog.Logger.Errorf("failed to remove iface, because %s", err.Error())
		return err
	}

	r.dotlog.Logger.Debugf("closed complete rcn")
	return err
}
