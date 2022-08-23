// Copyright (c) 2022 Notch Inc & AUTHORS All rights reserved.
// Use of this source code is governed by a BSD 3-Clause License
// license that can be found in the LICENSE file.

package rcnsock

// a package that communicates using rcn and unix sockets
//

import (
	"encoding/gob"
	"net"
	"os"

	"github.com/Notch-Technologies/dotshake/client/grpc"
	"github.com/Notch-Technologies/dotshake/dotlog"
)

type RcnSock struct {
	signalClient grpc.SignalClientImpl

	ip   string
	cidr string

	dotlog *dotlog.DotLog

	ch chan struct{}
}

// if scp is nil when making this function call, just listen
//
func NewRcnSock(
	dotlog *dotlog.DotLog,
	ch chan struct{},
) *RcnSock {
	return &RcnSock{

		dotlog: dotlog,

		ch: ch,
	}
}

func (s *RcnSock) cleanup() error {
	if _, err := os.Stat(sockaddr); err == nil {
		if err := os.RemoveAll(sockaddr); err != nil {
			return err
		}
	}
	return nil
}

func (s *RcnSock) listen(conn net.Conn) {
	defer conn.Close()

	decoder := gob.NewDecoder(conn)
	encoder := gob.NewEncoder(conn)

	for {
		mes := &RcnDialSock{}
		err := decoder.Decode(mes)
		if err != nil {
			break
		}

		switch mes.MessageType {
		case CompletedConn:
			status := s.signalClient.GetConnStatus()
			mes.DialDotshakeStatus.Ip = s.ip
			mes.DialDotshakeStatus.Cidr = s.cidr
			mes.DialDotshakeStatus.Status = status
		}

		err = encoder.Encode(mes)
		if err != nil {
			s.dotlog.Logger.Errorf("failed to encode wondersock. %s", err.Error())
			break
		}
	}
}

func (s *RcnSock) Connect(
	signalClient grpc.SignalClientImpl,
	ip, cidr string,
) error {
	err := s.cleanup()
	if err != nil {
		return err
	}

	s.ip = ip
	s.cidr = cidr
	s.signalClient = signalClient

	listener, err := net.Listen("unix", sockaddr)
	if err != nil {
		return err
	}

	go func() {
		<-s.ch
		s.dotlog.Logger.Debugf("close the rcn socket")
		s.cleanup()
	}()

	s.dotlog.Logger.Debugf("starting rcn socket")
	for {
		conn, err := listener.Accept()
		if err != nil {
			s.dotlog.Logger.Errorf("failed to accept rcn socket. %s", err.Error())
		}

		s.dotlog.Logger.Debugf("accepted rcn sock")

		go s.listen(conn)
	}
}

func (s *RcnSock) DialDotshakeStatus() (*DialDotshakeStatus, error) {
	conn, err := net.Dial("unix", sockaddr)
	defer conn.Close()
	if err != nil {
		return nil, err
	}

	decoder := gob.NewDecoder(conn)
	encoder := gob.NewEncoder(conn)

	d := &RcnDialSock{
		MessageType: CompletedConn,

		DialDotshakeStatus: &DialDotshakeStatus{},
	}

	err = encoder.Encode(d)
	if err != nil {
		return nil, err
	}

	err = decoder.Decode(d)
	if err != nil {
		return nil, err
	}

	return d.DialDotshakeStatus, nil
}
