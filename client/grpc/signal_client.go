// Copyright (c) 2022 Notch Inc & AUTHORS All rights reserved.
// Use of this source code is governed by a BSD 3-Clause License
// license that can be found in the LICENSE file.

package grpc

import (
	"context"
	"io"
	"sync"
	"time"

	"github.com/Notch-Technologies/client-go/notch/dotshake/v1/negotiation"
	"github.com/Notch-Technologies/client-go/notch/dotshake/v1/rtc"
	"github.com/Notch-Technologies/dotshake/dotlog"
	"github.com/Notch-Technologies/dotshake/rcn/conn"
	"github.com/Notch-Technologies/dotshake/utils"
	"github.com/pion/ice/v2"
	"google.golang.org/grpc"
	"google.golang.org/grpc/connectivity"
	"google.golang.org/grpc/metadata"
	"google.golang.org/protobuf/types/known/emptypb"
)

type SignalClientImpl interface {
	Candidate(dstmk, srcmk string, candidate ice.Candidate) error
	Offer(dstmk, srcmk string, uFlag string, pwd string) error
	Answer(dstmk, srcmk string, uFlag string, pwd string) error

	StartConnect(mk string, handler func(msg *negotiation.NegotiationRequest) error) error

	WaitStartConnect()
	IsReady() bool
	GetStunTurnConfig() (*rtc.GetStunTurnConfigResponse, error)

	// Status
	DisConnected() error
	Connected() error
	GetConnStatus() string
}

type SignalClient struct {
	negClient negotiation.NegotiationServiceClient
	rtcClient rtc.RtcServiceClient
	conn      *grpc.ClientConn

	ctx context.Context

	mux sync.Mutex

	connState *conn.ConnectState

	dotlog *dotlog.DotLog
}

func NewSignalClient(
	conn *grpc.ClientConn,
	cs *conn.ConnectState,
	dotlog *dotlog.DotLog,
) SignalClientImpl {
	return &SignalClient{
		negClient: negotiation.NewNegotiationServiceClient(conn),
		rtcClient: rtc.NewRtcServiceClient(conn),
		conn:      conn,
		ctx:       context.Background(),
		mux:       sync.Mutex{},
		// at this time, it is in an absolutely DISCONNECTED state
		connState: cs,
		dotlog:    dotlog,
	}
}

func (c *SignalClient) Candidate(dstmk, srcmk string, candidate ice.Candidate) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	msg := &negotiation.CandidateRequest{
		DstPeerMachineKey: dstmk,
		SrcPeerMachineKey: srcmk,
		Candidate:         candidate.Marshal(),
	}
	_, err := c.negClient.Candidate(ctx, msg)
	if err != nil {
		return err
	}

	return nil
}

func (c *SignalClient) Offer(
	dstmk, srcmk string,
	uFlag string,
	pwd string,
) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	msg := &negotiation.HandshakeRequest{
		DstPeerMachineKey: dstmk,
		SrcPeerMachineKey: srcmk,
		UFlag:             uFlag,
		Pwd:               pwd,
	}
	_, err := c.negClient.Offer(ctx, msg)
	if err != nil {
		return err
	}

	return nil
}

func (c *SignalClient) Answer(
	dstmk, srcmk string,
	uFlag string,
	pwd string,
) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	msg := &negotiation.HandshakeRequest{
		DstPeerMachineKey: dstmk,
		SrcPeerMachineKey: srcmk,
		UFlag:             uFlag,
		Pwd:               pwd,
	}
	_, err := c.negClient.Answer(ctx, msg)
	if err != nil {
		return err
	}

	return nil
}

// actually connected to grpc stream
func (c *SignalClient) connectStream(ctx context.Context) (negotiation.NegotiationService_StartConnectClient, error) {

	stream, err := c.negClient.StartConnect(ctx, grpc.WaitForReady(true))
	if err != nil {
		return nil, err
	}

	// set connState to Connected
	c.connState.Connected()

	return stream, nil
}

func (c *SignalClient) StartConnect(mk string, handler func(msg *negotiation.NegotiationRequest) error) error {
	md := metadata.New(map[string]string{utils.MachineKey: mk})
	ctx := metadata.NewOutgoingContext(c.ctx, md)

	stream, err := c.connectStream(ctx)
	if err != nil {
		return err
	}

	defer func() {
		err := stream.CloseSend()
		if err != nil {
			c.dotlog.Logger.Errorf("failed to close start connect")
			return
		}
	}()

	for {
		msg, err := stream.Recv()
		if err == io.EOF {
			c.dotlog.Logger.Errorf("connect stream return to EOF, received by [%s]", mk)
			return err
		}
		if err != nil {
			c.dotlog.Logger.Errorf("failed to get grpc client stream for machinek key: %s", msg.DstPeerMachineKey)
			return err
		}

		err = handler(msg)
		if err != nil {
			c.dotlog.Logger.Errorf("failed to handle grpc client stream stream in machine key: %s", msg.DstPeerMachineKey)
			return err
		}
	}
}

// function to wait until connState becomes Connected until it can
//
func (c *SignalClient) WaitStartConnect() {
	if c.connState.IsConnected() {
		return
	}

	ch := c.connState.GetConnChan()
	select {
	case <-c.ctx.Done():
	case <-ch:
	}
}

func (c *SignalClient) IsReady() bool {
	return c.conn.GetState() == connectivity.Ready || c.conn.GetState() == connectivity.Idle
}

func (c *SignalClient) GetStunTurnConfig() (*rtc.GetStunTurnConfigResponse, error) {
	conf, err := c.rtcClient.GetStunTurnConfig(c.ctx, &emptypb.Empty{})
	if err != nil {
		return nil, err
	}

	return conf, nil
}

func (c *SignalClient) DisConnected() error {
	c.connState.DisConnected()
	return nil
}

func (c *SignalClient) Connected() error {
	c.connState.Connected()
	return nil
}

func (c *SignalClient) GetConnStatus() string {
	c.connState.Connected()
	status := c.connState.GetConnStatus()
	return status.String()
}
