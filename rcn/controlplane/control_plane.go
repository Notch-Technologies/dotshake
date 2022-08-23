// Copyright (c) 2022 Notch Inc & AUTHORS All rights reserved.
// Use of this source code is governed by a BSD 3-Clause License
// license that can be found in the LICENSE file.

package controlplane

// this package is responsible for communication with the signal server
// it also has the structure of ice of the remote peer as a map key with the machine key of the remote peer
// when the communication with the signal server is performed and operations are performed on the peer, they will basically be performed here.
//

import (
	"errors"
	"strings"
	"sync"
	"time"

	"github.com/Notch-Technologies/client-go/notch/dotshake/v1/machine"
	"github.com/Notch-Technologies/client-go/notch/dotshake/v1/negotiation"
	"github.com/Notch-Technologies/dotshake/client/grpc"
	"github.com/Notch-Technologies/dotshake/conf"
	"github.com/Notch-Technologies/dotshake/dotlog"
	"github.com/Notch-Technologies/dotshake/rcn/rcnsock"
	"github.com/Notch-Technologies/dotshake/rcn/webrtc"
	"github.com/Notch-Technologies/dotshake/wireguard"
	"github.com/pion/ice/v2"
	"golang.zx2c4.com/wireguard/wgctrl/wgtypes"
)

type ControlPlane struct {
	signalClient grpc.SignalClientImpl
	serverClient grpc.ServerClientImpl

	sock *rcnsock.RcnSock

	peerConns  map[string]*webrtc.Ice //  with ice structure per clientmachinekey
	mk         string
	clientConf *conf.ClientConf
	stconf     *webrtc.StunTurnConfig

	mu                  *sync.Mutex
	ch                  chan struct{}
	waitForRemoteConnCh chan *webrtc.Ice

	dotlog *dotlog.DotLog
}

func NewControlPlane(
	signalClient grpc.SignalClientImpl,
	serverClient grpc.ServerClientImpl,
	sock *rcnsock.RcnSock,
	mk string,
	clientConf *conf.ClientConf,
	ch chan struct{},
	dotlog *dotlog.DotLog,
) *ControlPlane {
	return &ControlPlane{
		signalClient: signalClient,
		serverClient: serverClient,

		sock: sock,

		peerConns:  make(map[string]*webrtc.Ice),
		mk:         mk,
		clientConf: clientConf,

		mu:                  &sync.Mutex{},
		ch:                  ch,
		waitForRemoteConnCh: make(chan *webrtc.Ice),

		dotlog: dotlog,
	}
}

func (c *ControlPlane) parseStun(url, uname, pw string) (*ice.URL, error) {
	stun, err := ice.ParseURL(url)
	if err != nil {
		return nil, err
	}

	stun.Username = uname
	stun.Password = pw
	return stun, err
}

func (c *ControlPlane) parseTurn(url, uname, pw string) (*ice.URL, error) {
	turn, err := ice.ParseURL(url)
	if err != nil {
		return nil, err
	}
	turn.Username = uname
	turn.Password = pw

	return turn, err
}

// set stun turn url to use webrtc
// (shinta) be sure to call this function before using the ConnectSignalServer
//
func (c *ControlPlane) ConfigureStunTurnConf() error {
	conf, err := c.signalClient.GetStunTurnConfig()
	if err != nil {
		// TOOD: (shinta) retry
		return err
	}

	stun, err := c.parseStun(
		conf.RtcConfig.StunHost.Url,
		conf.RtcConfig.TurnHost.Username,
		conf.RtcConfig.TurnHost.Password,
	)
	if err != nil {
		return err
	}

	turn, err := c.parseTurn(
		conf.RtcConfig.TurnHost.Url,
		conf.RtcConfig.TurnHost.Username,
		conf.RtcConfig.TurnHost.Password,
	)
	if err != nil {
		return err
	}

	stcof := webrtc.NewStunTurnConfig(stun, turn)

	c.stconf = stcof

	return nil
}

func (c *ControlPlane) receiveSignalingProcess(
	remotemk string,
	msgType negotiation.NegotiationType,
	peer *webrtc.Ice,
	uname string,
	pwd string,
	candidate string,
) error {
	switch msgType {
	case negotiation.NegotiationType_ANSWER:
		c.dotlog.Logger.Debugf("[%s] is sending answer to [%s]", peer.GetLocalMachineKey(), peer.GetRemoteMachineKey())
		peer.SendRemoteAnswerCh(remotemk, uname, pwd)
	case negotiation.NegotiationType_OFFER:
		c.dotlog.Logger.Debugf("[%s] is sending offer to [%s]", peer.GetLocalMachineKey(), peer.GetRemoteMachineKey())
		peer.SendRemoteOfferCh(remotemk, uname, pwd)
	case negotiation.NegotiationType_CANDIDATE:
		c.dotlog.Logger.Debugf("[%s] is sending candidate to [%s]", peer.GetLocalMachineKey(), peer.GetRemoteMachineKey())
		candidate, err := ice.UnmarshalCandidate(candidate)
		if err != nil {
			c.dotlog.Logger.Errorf("can not unmarshal candidate => [%s]", candidate)
			return err
		}
		peer.SendRemoteCandidate(candidate)
	}

	return nil
}

// through StartConnect, the results of the execution of functions such as
// candidate required for udp hole punching are received from the dotengine side
//
func (c *ControlPlane) ConnectSignalServer() {
	go func() {
		err := c.signalClient.StartConnect(c.mk, func(res *negotiation.NegotiationRequest) error {
			c.mu.Lock()
			defer c.mu.Unlock()

			dstPeerMachineKey := res.GetDstPeerMachineKey()
			if dstPeerMachineKey == "" {
				c.dotlog.Logger.Errorf("empty dst peer machine key")
				return errors.New("empty dst peer machine key")
			}

			peer := c.peerConns[res.GetDstPeerMachineKey()]

			// for initial offer
			if peer == nil {
				var err error
				peer, err = c.initialOfferForRemotePeer(dstPeerMachineKey)
				if err != nil {
					c.dotlog.Logger.Errorf("empty remote peer connection, dst remote peer machine key is [%s]", dstPeerMachineKey)
					return err
				}
			}

			err := c.receiveSignalingProcess(
				res.GetDstPeerMachineKey(),
				res.GetType(),
				peer,
				res.GetUFlag(),
				res.GetPwd(),
				res.GetCandidate(),
			)
			if err != nil {
				return err
			}
			return nil
		})
		if err != nil {
			close(c.ch)
			return
		}
	}()
	c.signalClient.WaitStartConnect()
}

func (c *ControlPlane) initialOfferForRemotePeer(dstPeerMk string) (*webrtc.Ice, error) {
	c.dotlog.Logger.Debugf("initial connection for [%s]", dstPeerMk)

	res, err := c.serverClient.SyncRemoteMachinesConfig(c.mk)
	if err != nil {
		return nil, err
	}

	for _, rp := range res.GetRemotePeers() {
		if rp.RemoteClientMachineKey != dstPeerMk {
			continue
		}

		i, err := c.configureIce(rp, res.Ip, res.Cidr)
		if err != nil {
			return nil, err
		}

		c.peerConns[dstPeerMk] = i
		c.waitForRemoteConnCh <- i
		return c.peerConns[dstPeerMk], nil
	}

	// (shinta) is it inherently impossible?
	return nil, errors.New("failed to initial offer")
}

// keep the latest state of Peers received from the server
//
func (c *ControlPlane) syncRemotePeerConfig(remotePeers []*machine.RemotePeer) error {
	remotePeerMap := make(map[string]struct{})
	for _, p := range remotePeers {
		remotePeerMap[p.GetRemoteClientMachineKey()] = struct{}{}
	}

	unnecessary := []string{}
	for p := range c.peerConns {
		if _, ok := remotePeerMap[p]; !ok {
			unnecessary = append(unnecessary, p)
		}
	}

	if len(unnecessary) == 0 {
		return nil
	}

	for _, p := range unnecessary {
		conn, exists := c.peerConns[p]
		if exists {
			delete(c.peerConns, p)
			conn.Cleanup()
		}
		c.dotlog.Logger.Debugf("there are no peers, even though there should be")
	}

	c.dotlog.Logger.Debugf("completed peersConn delete in signal control plane => %v", unnecessary)
	return nil
}

func (c *ControlPlane) configureIce(peer *machine.RemotePeer, myip, mycidr string) (*webrtc.Ice, error) {
	k, err := wgtypes.ParseKey(c.clientConf.WgPrivateKey)
	if err != nil {
		return nil, err
	}

	var pk string
	if c.clientConf.PreSharedKey != "" {
		k, err := wgtypes.ParseKey(c.clientConf.PreSharedKey)
		if err != nil {
			return nil, err
		}
		pk = k.String()
	}

	remoteip := strings.Join(peer.GetAllowedIPs(), ",")
	i := webrtc.NewIce(
		c.signalClient,

		c.sock,

		peer.RemoteWgPubKey,
		remoteip,
		peer.GetRemoteClientMachineKey(),

		myip,
		mycidr,
		k,
		wireguard.WgPort,
		c.clientConf.TunName,
		pk,
		c.mk,

		c.stconf,
		c.clientConf.BlackList,

		c.dotlog,
		c.ch,
	)

	return i, nil
}

func (c *ControlPlane) NotifyRemotePeersConn(connPeers []*machine.RemotePeer, ip, cidr string) error {
	for _, p := range connPeers {
		c.dotlog.Logger.Debugf("wanna connect to remote machine => [%s]", p.GetRemoteClientMachineKey())

		rmk := p.GetRemoteClientMachineKey()
		_, ok := c.peerConns[rmk]

		if !ok {
			i, err := c.configureIce(p, ip, cidr)
			if err != nil {
				return err
			}

			c.peerConns[rmk] = i
			c.waitForRemoteConnCh <- i
			continue
		}

		c.dotlog.Logger.Debugf("[%s] has [%s] peer connection,", c.mk, rmk)

		return nil
	}
	return nil
}

func (c *ControlPlane) isExistPeer(remoteMachineKey string) bool {
	_, exist := c.peerConns[remoteMachineKey]
	return exist
}

// function to wait until the channel is sent from SetupRemotePeerConn to waitForRemoteConnCh
//
func (c *ControlPlane) WaitForRemoteConn() {
	for {
		select {
		case ice := <-c.waitForRemoteConnCh:
			if !c.signalClient.IsReady() || !c.isExistPeer(ice.GetRemoteMachineKey()) {
				c.dotlog.Logger.Errorf("signal client is not available, execute loop. applicable remote peer => [%s]", ice.GetRemoteMachineKey())
				continue
			}

			c.dotlog.Logger.Debugf("starting gathering process for remote machine => [%s]", ice.GetRemoteMachineKey())

			err := ice.ConfigureGatherProcess()
			if err != nil {
				c.dotlog.Logger.Errorf("failed to configure gathering process for [%s]", ice.GetRemoteMachineKey())
				continue
			}

			err = ice.StartGatheringProcess()
			if err != nil {
				c.dotlog.Logger.Errorf("failed to start gathering process for [%s]", ice.GetRemoteMachineKey())
				continue
			}
		}
	}
}

// ConnectToHangoutMachines to keep the Peer's information up to date asynchronously
// notify here when another machine or itself joinsHangOutMachines
// when coming adding new peer or initial sync
//
func (c *ControlPlane) StartHangoutMachines() {
	go func() {
		c.serverClient.ConnectToHangoutMachines(c.mk, func(res *machine.HangOutMachinesResponse) error {
			c.mu.Lock()
			defer c.mu.Unlock()

			if res.GetRemotePeers() != nil {
				c.dotlog.Logger.Debugf("got remote peers => %v", res.GetRemotePeers())
				err := c.syncRemotePeerConfig(res.GetRemotePeers())
				if err != nil {
					return err
				}
			}

			// TODO: (shinta) it seems a little confusing. will refactoring. https://github.com/Notch-Technologies/dotshake/issues/21
			// initialize to maintain agent integrity when a disconnected Machine reconnects
			//
			if res.GetHangOutType() == machine.HangOutType_DISCONNECT {
				if peer, ok := c.peerConns[res.TargetMachineKey]; ok {
					err := peer.Setup()
					if err != nil {
						c.dotlog.Logger.Errorf("failed to resetup %s, %s", res.TargetMachineKey, err.Error())
					}
				}
				return nil
			}

			err := c.NotifyRemotePeersConn(res.GetRemotePeers(), res.Ip, res.Cidr)
			if err != nil {
				return err
			}

			return nil
		})
	}()
}

// maintain flexible connections by updating remote machines
// information on a regular basis, rather than only when other Machines join
//
func (c *ControlPlane) SyncRemoteMachine() error {
	ticker := time.NewTicker(1 * time.Minute)
	for {
		select {
		case <-ticker.C:
			res, err := c.serverClient.SyncRemoteMachinesConfig(c.mk)
			if err != nil {
				return err
			}

			// TODO: (shinta) compare with existing c.peerConns and update only when there is a difference?
			// maybe will be good perfomance
			if res.GetRemotePeers() != nil {
				err := c.syncRemotePeerConfig(res.GetRemotePeers())
				if err != nil {
					c.dotlog.Logger.Errorf("failed to sync remote peer config")
					return err
				}
			}
		}
	}
}

func (c *ControlPlane) Close() error {
	for mk, ice := range c.peerConns {
		if ice == nil {
			continue
		}

		err := ice.Cleanup()
		if err != nil {
			return err
		}

		c.dotlog.Logger.Debugf("close the %s", mk)
	}

	c.dotlog.Logger.Debugf("finished in closing the control plane")

	return nil
}
