// Copyright (c) 2022 Notch Inc & AUTHORS All rights reserved.
// Use of this source code is governed by a BSD 3-Clause License
// license that can be found in the LICENSE file.

package webrtc

// ice and provides webrtc functionalit
// ice initializes one structure per remote machine key
//

import (
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/Notch-Technologies/dotshake/client/grpc"
	"github.com/Notch-Technologies/dotshake/dotlog"
	"github.com/Notch-Technologies/dotshake/iface"
	"github.com/Notch-Technologies/dotshake/rcn/conn"
	"github.com/Notch-Technologies/dotshake/rcn/proxy"
	"github.com/Notch-Technologies/dotshake/rcn/rcnsock"
	"github.com/pion/ice/v2"
	"golang.zx2c4.com/wireguard/wgctrl/wgtypes"
)

type Ice struct {
	signalClient grpc.SignalClientImpl

	sock *rcnsock.RcnSock

	sigexec *SigExecuter

	conn *conn.Conn

	wireproxy *proxy.WireProxy

	// channel to use when making a peer connection
	remoteOfferCh  chan Credentials
	remoteAnswerCh chan Credentials

	agent           *ice.Agent
	udpMux          *ice.UDPMuxDefault
	udpMuxSrflx     *ice.UniversalUDPMuxDefault
	udpMuxConn      *net.UDPConn
	udpMuxConnSrflx *net.UDPConn

	stunTurn *StunTurnConfig

	// remote
	remoteWgPubKey   string
	remoteIp         string
	remoteMachineKey string

	// local
	wgPubKey     string
	wgPrivKey    wgtypes.Key
	wgIface      string
	wgPort       int
	preSharedKey string

	// for iface
	ip   string
	cidr string

	mk string

	blackList []string

	mu      *sync.Mutex
	closeCh chan struct{}

	failedTimeout *time.Duration

	dotlog *dotlog.DotLog
}

func NewIce(
	signalClient grpc.SignalClientImpl,

	sock *rcnsock.RcnSock,

	// remote
	remoteWgPubKey string,
	remoteip string,
	remoteMachineKey string,

	// yours
	ip string,
	cidr string,
	wgPrivateKey wgtypes.Key,
	wgPort int,
	wgIface string,
	presharedKey string,
	mk string,

	stunTurn *StunTurnConfig,
	blacklist []string,

	dotlog *dotlog.DotLog,

	closeCh chan struct{},
) *Ice {
	failedtimeout := time.Second * 5
	return &Ice{
		signalClient: signalClient,

		sock: sock,

		remoteOfferCh:  make(chan Credentials),
		remoteAnswerCh: make(chan Credentials),

		stunTurn: stunTurn,

		remoteWgPubKey:   remoteWgPubKey,
		remoteIp:         remoteip,
		remoteMachineKey: remoteMachineKey,

		wgPubKey:     wgPrivateKey.PublicKey().String(),
		wgPrivKey:    wgPrivateKey,
		wgIface:      wgIface,
		wgPort:       wgPort,
		preSharedKey: presharedKey,
		ip:           ip,
		cidr:         cidr,
		mk:           mk,

		blackList: blacklist,

		mu:      &sync.Mutex{},
		closeCh: closeCh,

		failedTimeout: &failedtimeout,

		dotlog: dotlog,
	}
}

// must be called before calling ConfigureGatherProcess
//
func (i *Ice) Setup() (err error) {
	i.mu.Lock()
	defer i.mu.Unlock()

	// configure sigexe
	//
	se := NewSigExecuter(i.signalClient, i.remoteMachineKey, i.mk, i.dotlog)
	i.sigexec = se

	// configure ice agent
	i.udpMuxConn, err = net.ListenUDP("udp4", &net.UDPAddr{Port: 0})
	i.udpMuxConnSrflx, err = net.ListenUDP("udp4", &net.UDPAddr{Port: 0})

	i.udpMux = ice.NewUDPMuxDefault(ice.UDPMuxParams{UDPConn: i.udpMuxConn})
	i.udpMuxSrflx = ice.NewUniversalUDPMuxDefault(ice.UniversalUDPMuxParams{UDPConn: i.udpMuxConnSrflx})

	i.agent, err = ice.NewAgent(&ice.AgentConfig{
		MulticastDNSMode: ice.MulticastDNSModeDisabled,
		NetworkTypes:     []ice.NetworkType{ice.NetworkTypeUDP4},
		Urls:             i.stunTurn.GetStunTurnsURL(),
		CandidateTypes:   []ice.CandidateType{ice.CandidateTypeHost, ice.CandidateTypeServerReflexive, ice.CandidateTypeRelay},
		FailedTimeout:    i.failedTimeout,
		InterfaceFilter:  i.getBlackListWithInterfaceFilter(),
		UDPMux:           i.udpMux,
		UDPMuxSrflx:      i.udpMuxSrflx,
	})
	if err != nil {
		return err
	}

	// configure ice candidate functions
	err = i.agent.OnCandidate(i.sigexec.Candidate)
	if err != nil {
		return err
	}

	err = i.agent.OnConnectionStateChange(i.IceConnectionHasBeenChanged)
	if err != nil {
		return err
	}

	err = i.agent.OnSelectedCandidatePairChange(i.IceSelectedHasCandidatePairChanged)
	if err != nil {
		return err
	}

	// configure iface
	iface := iface.NewIface(i.wgIface, i.wgPrivKey.String(), i.ip, i.cidr, i.dotlog)

	// configure wire proxy
	wireproxy := proxy.NewWireProxy(
		iface,
		i.remoteWgPubKey,
		i.remoteIp,
		i.wgIface,
		fmt.Sprintf("127.0.0.1:%d", i.wgPort),
		i.preSharedKey,
		i.dotlog,
		i.agent,
	)

	i.wireproxy = wireproxy

	return nil
}

// TODO: (shinta)
// more detailed handling is needed.
// by handling failures, we need to establish a connection path using DoubleNat? or
// Ether(call me ????????????) when a connection cannot be made.
func (i *Ice) IceConnectionHasBeenChanged(state ice.ConnectionState) {
	switch state {
	case ice.ConnectionStateNew: // ConnectionStateNew ICE agent is gathering addresses
		i.dotlog.Logger.Infof("new connections collected, [%s]", state.String())
	case ice.ConnectionStateChecking: // ConnectionStateNew ICE agent is gathering addresses
		i.dotlog.Logger.Infof("checking agent state, [%s]", state.String())
	case ice.ConnectionStateConnected: // ConnectionStateConnected ICE agent has a pairing, but is still checking other pairs
		i.dotlog.Logger.Debugf("agent [%s]", state.String())
	case ice.ConnectionStateCompleted: // ConnectionStateConnected ICE agent has a pairing, but is still checking other pairs
		err := i.signalClient.Connected()
		if err != nil {
			i.dotlog.Logger.Errorf("the agent connection was successful but I received an error in the function that updates the status to connect, [%s]", state.String())
		}
		i.dotlog.Logger.Debugf("successfully connected to agent, [%s]", state.String())
	case ice.ConnectionStateFailed: // ConnectionStateFailed ICE agent never could successfully connect
		err := i.signalClient.DisConnected()
		if err != nil {
			i.dotlog.Logger.Errorf("agent connection failed, but failed to set the connection state to disconnect, [%s]", state.String())
		}
	case ice.ConnectionStateDisconnected: // ConnectionStateDisconnected ICE agent connected successfully, but has entered a failed state
		err := i.signalClient.DisConnected()
		if err != nil {
			i.dotlog.Logger.Errorf("agent connected successfully, but has entered a failed state, [%s]", state.String())
		}
	case ice.ConnectionStateClosed: // ConnectionStateClosed ICE agent has finished and is no longer handling requests
		i.dotlog.Logger.Infof("agent has finished and is no longer handling requests, [%s]", state.String())
	}
}

func (i *Ice) IceSelectedHasCandidatePairChanged(local ice.Candidate, remote ice.Candidate) {
	i.dotlog.Logger.Infof("[CANDIDATE COMPLETED] agent candidates were found, local:[%s] <-> remote:[%s]", local.Address(), remote.Address())
}

// be sure to read this function before using the Ice structures
//
func (i *Ice) ConfigureGatherProcess() error {
	err := i.Setup()
	if err != nil {
		i.dotlog.Logger.Errorf("failed to configure gather process")
		return err
	}

	return nil
}

func (i *Ice) GetRemoteMachineKey() string {
	return i.remoteMachineKey
}

func (i *Ice) GetLocalMachineKey() string {
	return i.mk
}

func (i *Ice) getBlackListWithInterfaceFilter() func(string) bool {
	var blackListMap map[string]struct{}
	if i.blackList != nil {
		blackListMap = make(map[string]struct{})
		for _, s := range i.blackList {
			blackListMap[s] = struct{}{}
		}
	}

	return func(iFace string) bool {
		if len(blackListMap) == 0 {
			return true
		}
		_, ok := blackListMap[iFace]
		return !ok
	}
}

func (i *Ice) closeIceAgent() error {
	i.dotlog.Logger.Debugf("starting close ice agent process")

	i.mu.Lock()
	defer i.mu.Unlock()

	err := i.udpMux.Close()
	if err != nil {
		return err
	}

	err = i.udpMuxSrflx.Close()
	if err != nil {
		return err
	}

	err = i.udpMuxConn.Close()
	if err != nil {
		return err
	}

	err = i.udpMuxConnSrflx.Close()
	if err != nil {
		return err
	}

	err = i.agent.Close()
	if err != nil {
		return err
	}

	i.signalClient.DisConnected()

	i.dotlog.Logger.Debugf("completed clean ice agent process")

	return nil
}

func (i *Ice) getLocalUserIceAgentCredentials() (string, string, error) {
	uname, pwd, err := i.agent.GetLocalUserCredentials()
	if err != nil {
		return "", "", err
	}

	return uname, pwd, nil
}

// be sure to read ConfigureGatherProcess before calling this function
//
func (i *Ice) StartGatheringProcess() error {
	go i.waitingForSignalProcess()

	err := i.signalOffer()
	if err != nil {
		i.dotlog.Logger.Errorf("failed to signal offer, because %s", err.Error())
		return err
	}

	return nil
}

func (i *Ice) startConn(uname, pwd string) error {
	i.conn = conn.NewConn(
		i.agent,
		uname,
		pwd,
		i.wireproxy,
		i.remoteWgPubKey,
		i.wgPubKey,
		i.dotlog,
	)

	err := i.conn.Start()
	if err != nil {
		return err
	}

	return nil
}

func (i *Ice) CloseConn() error {
	if i.conn != nil {
		err := i.conn.Close()
		if err != nil {
			return err
		}
	}

	return nil
}

func (i *Ice) Cleanup() error {
	if i.conn != nil {
		err := i.conn.Close()
		if err != nil {
			return err
		}
	}

	err := i.CloseIce()
	if err != nil {
		return err
	}

	return nil
}

func (i *Ice) CloseIce() error {
	err := i.closeIceAgent()
	if err != nil {
		return err
	}

	return nil
}

func (i *Ice) waitingForSignalProcess() {
	var credentials Credentials

	for {
		select {
		case credentials = <-i.remoteAnswerCh:
			i.dotlog.Logger.Debugf("receive credentials from [%s]", i.remoteMachineKey)
		case credentials = <-i.remoteOfferCh:
			i.dotlog.Logger.Debugf("receive offer from [%s]", i.remoteMachineKey)
			err := i.signalAnswer()
			if err != nil {
				i.dotlog.Logger.Errorf("failed to signal offer, %s", err.Error())
			}
		}

		err := i.agent.GatherCandidates()
		if err != nil {
			i.dotlog.Logger.Errorf("failed to gather candidates, %s", err.Error())
			return
		}

		err = i.startConn(credentials.UserName, credentials.Pwd)
		if err != nil {
			i.dotlog.Logger.Errorf("failed to start conn, %s", err.Error())
			return
		}

		i.ConnectSock()
	}
}

func (i *Ice) ConnectSock() {
	go func() {
		err := i.sock.Connect(i.signalClient, i.ip, i.cidr)
		if err != nil {
			i.dotlog.Logger.Errorf("failed connect rcn sock, %s", err.Error())
		}
		i.dotlog.Logger.Debugf("rcn sock connect has been disconnected")
	}()
}

func (i *Ice) signalAnswer() error {
	i.mu.Lock()
	defer i.mu.Unlock()

	uname, pwd, err := i.getLocalUserIceAgentCredentials()
	if err != nil {
		return err
	}

	err = i.sigexec.Answer(uname, pwd)
	if err != nil {
		return err
	}

	i.dotlog.Logger.Debugf("answer has been sent to the signal server")

	return nil
}

func (i *Ice) signalOffer() error {
	i.mu.Lock()
	defer i.mu.Unlock()

	uname, pwd, err := i.getLocalUserIceAgentCredentials()
	if err != nil {
		return err
	}

	err = i.sigexec.Offer(uname, pwd)
	if err != nil {
		return err
	}

	return nil
}

func (i *Ice) SendRemoteOfferCh(remotemk, uname, pwd string) {
	select {
	case i.remoteOfferCh <- *NewCredentials(uname, pwd):
		i.dotlog.Logger.Debugf("send offer to [%s]", remotemk)
	default:
		i.dotlog.Logger.Debugf("%s agent waitForSignalingProcess does not seem to have been started", remotemk)
	}
}

func (i *Ice) SendRemoteAnswerCh(remotemk, uname, pwd string) {
	select {
	case i.remoteAnswerCh <- *NewCredentials(uname, pwd):
		i.dotlog.Logger.Debugf("send answer to [%s]", remotemk)
	default:
		i.dotlog.Logger.Debugf("answer skipping message to %s", remotemk)
	}
}

func (i *Ice) SendRemoteCandidate(candidate ice.Candidate) {
	go func() {
		i.mu.Lock()
		defer i.mu.Unlock()

		if i.agent == nil {
			i.dotlog.Logger.Errorf("agent is nil")
			return
		}

		err := i.agent.AddRemoteCandidate(candidate)
		if err != nil {
			i.dotlog.Logger.Errorf("cannot add remote candidate")
			return
		}

		i.dotlog.Logger.Debugf("send candidate to [%s]", i.remoteMachineKey)
	}()
}
