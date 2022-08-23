// Copyright (c) 2022 Notch Inc & AUTHORS All rights reserved.
// Use of this source code is governed by a BSD 3-Clause License
// license that can be found in the LICENSE file.

package proxy

import (
	"context"
	"net"

	"github.com/Notch-Technologies/dotshake/dotlog"
	"github.com/Notch-Technologies/dotshake/iface"
	"github.com/Notch-Technologies/dotshake/wireguard"
	"github.com/pion/ice/v2"
)

type WireProxy struct {
	iface *iface.Iface

	// proxy config
	remoteWgPubKey string // remote peer wg pub key
	remoteIp       string // remote peer ip
	wgIface        string // your wg iface
	listenAddr     string // proxy addr
	preSharedKey   string // your preshared key

	remoteConn net.Conn
	localConn  net.Conn

	agent *ice.Agent

	// localProxyBuffer  []byte
	// remoteProxyBuffer []byte

	ctx        context.Context
	cancelFunc context.CancelFunc

	dotlog *dotlog.DotLog
}

// TODO: (shinta) rewrite to proxy using sock5?
func NewWireProxy(
	iface *iface.Iface,
	remoteWgPubKey string,
	remoteip string,
	wgiface string,
	listenAddr string,
	presharedkey string,
	dotlog *dotlog.DotLog,
	agent *ice.Agent,
) *WireProxy {
	ctx, cancel := context.WithCancel(context.Background())

	return &WireProxy{
		iface: iface,

		remoteWgPubKey: remoteWgPubKey,
		remoteIp:       remoteip,

		wgIface:      wgiface,
		listenAddr:   listenAddr,
		preSharedKey: presharedkey,

		// localProxyBuffer:  make([]byte, 1500),
		// remoteProxyBuffer: make([]byte, 1500),

		agent: agent,

		ctx:        ctx,
		cancelFunc: cancel,

		dotlog: dotlog,
	}
}

func (w *WireProxy) setup(remote *ice.Conn) error {
	w.remoteConn = remote
	udpConn, err := net.Dial("udp", w.listenAddr)
	if err != nil {
		return err
	}
	w.localConn = udpConn

	return nil
}

func (w *WireProxy) configureNoProxy() error {
	w.dotlog.Logger.Debugf("using no proxy")

	udpAddr, err := net.ResolveUDPAddr("udp", w.remoteConn.RemoteAddr().String())
	if err != nil {
		return err
	}
	udpAddr.Port = wireguard.WgPort

	err = w.iface.ConfigureToRemotePeer(
		w.remoteWgPubKey,
		w.remoteIp,
		udpAddr,
		wireguard.DefaultWgKeepAlive,
		w.preSharedKey,
	)
	if err != nil {
		w.dotlog.Logger.Errorf("failed to configure remote peer, %s", err.Error())
		return err
	}

	return nil

}

func (w *WireProxy) configureWireProxy() error {
	w.dotlog.Logger.Debugf("using wire proxy")

	udpAddr, err := net.ResolveUDPAddr(w.localConn.LocalAddr().Network(), w.localConn.LocalAddr().String())
	if err != nil {
		return err
	}

	err = w.iface.ConfigureToRemotePeer(
		w.remoteWgPubKey,
		w.remoteIp,
		udpAddr,
		wireguard.DefaultWgKeepAlive,
		w.preSharedKey,
	)
	if err != nil {
		w.dotlog.Logger.Errorf("failed to configure remote peer, %s", err.Error())
		return err
	}

	return nil
}

func (w *WireProxy) Stop() error {
	w.cancelFunc()

	if w.localConn == nil {
		w.dotlog.Logger.Errorf("error is unexpected, you are most likely referring to locallConn without calling the setup function")
		return nil
	}

	err := w.iface.RemoveRemotePeer(w.wgIface, w.remoteIp, w.remoteWgPubKey)
	if err != nil {
		return err
	}

	return nil
}

func shouldUseProxy(pair *ice.CandidatePair) bool {
	remoteIP := net.ParseIP(pair.Remote.Address())
	myIp := net.ParseIP(pair.Local.Address())
	remoteIsPublic := IsPublicIP(remoteIP)
	myIsPublic := IsPublicIP(myIp)

	if remoteIsPublic && pair.Remote.Type() == ice.CandidateTypeHost {
		return false
	}
	if myIsPublic && pair.Local.Type() == ice.CandidateTypeHost {
		return false
	}

	if pair.Local.Type() == ice.CandidateTypeHost && pair.Remote.Type() == ice.CandidateTypeHost {
		if !remoteIsPublic && !myIsPublic {
			return false
		}
	}

	return true
}

func IsPublicIP(ip net.IP) bool {
	if ip.IsLoopback() || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() || ip.IsPrivate() {
		return false
	}
	return true
}

func (w *WireProxy) StartProxy(remote *ice.Conn) error {
	err := w.setup(remote)
	if err != nil {
		return err
	}

	pair, err := w.agent.GetSelectedCandidatePair()
	if err != nil {
		return err
	}

	// TODO (shinta) refactor
	if shouldUseProxy(pair) {
		err = w.configureWireProxy()
		if err != nil {
			return err
		}
		w.startMon()

		return nil
	}

	err = w.configureNoProxy()
	if err != nil {
		return err
	}

	w.startMon()

	return nil
}

func (w *WireProxy) startMon() {
	w.dotlog.Logger.Debugf("starting monitoring proxy")
	go w.monLocalToRemoteProxy()
	go w.monRemoteToLocalProxy()
}

func (w *WireProxy) monLocalToRemoteProxy() {
	buf := make([]byte, 1500)
	for {
		select {
		case <-w.ctx.Done():
			return
		default:
			n, err := w.localConn.Read(buf)
			if err != nil {
				w.dotlog.Logger.Errorf("localConn cannot read remoteProxyBuffer [%s], size is %d", string(buf), n)
				continue
			}

			_, err = w.remoteConn.Write(buf[:n])
			if err != nil {
				w.dotlog.Logger.Errorf("localConn cannot write remoteProxyBuffer [%s], size is %d", string(buf), n)
				continue
			}

			// TODO: gathering buffer with dotmon
			// w.dotlog.Logger.Debugf("remoteConn read remoteProxyBuffer [%s]", w.remoteProxyBuffer[:n])
		}
	}
}

func (w *WireProxy) monRemoteToLocalProxy() {
	buf := make([]byte, 1500)
	for {
		select {
		case <-w.ctx.Done():
			w.dotlog.Logger.Errorf("close the local proxy. close the remote ip here [%s], ", w.remoteIp)
			return
		default:
			n, err := w.remoteConn.Read(buf)
			if err != nil {
				w.dotlog.Logger.Errorf("remoteConn cannot read localProxyBuffer [%s], size is %d", string(buf), n)
				continue
			}

			_, err = w.localConn.Write(buf[:n])
			if err != nil {
				w.dotlog.Logger.Errorf("localConn cannot write localProxyBuffer [%s], size is %d", string(buf), n)
				continue
			}

			// TODO: gathering buffer with dotmon
			// w.dotlog.Logger.Debugf("localConn read localProxyBuffer [%s]", w.localProxyBuffer[:n])
		}
	}
}
