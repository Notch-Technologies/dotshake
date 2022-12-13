// Copyright (c) 2022 Notch Inc & AUTHORS All rights reserved.
// Use of this source code is governed by a BSD 3-Clause License
// license that can be found in the LICENSE file.

package grpc

import (
	"context"

	"github.com/Notch-Technologies/client-go/notch/dotshake/v1/login_session"
	"github.com/Notch-Technologies/client-go/notch/dotshake/v1/machine"
	"github.com/Notch-Technologies/dotshake/dotlog"
	"github.com/Notch-Technologies/dotshake/system"
	"github.com/Notch-Technologies/dotshake/utils"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
	"google.golang.org/protobuf/types/known/emptypb"
)

type ServerClientImpl interface {
	GetMachine(mk, wgPubKey string) (*machine.GetMachineResponse, error)

	SyncRemoteMachinesConfig(mk string) (*machine.SyncMachinesResponse, error)

	// ConnectToHangoutMachines(mk string, handler func(msg *machine.HangOutMachinesResponse) error) error
	JoinHangoutMachines(mk string) (*machine.HangOutMachinesResponse, error)

	ConnectStreamPeerLoginSession(mk string) (*login_session.PeerLoginSessionResponse, error)
}

type ServerClient struct {
	machineClient      machine.MachineServiceClient
	loginSessionClient login_session.LoginSessionServiceClient
	conn               *grpc.ClientConn
	ctx                context.Context
	dotlog             *dotlog.DotLog
}

func NewServerClient(
	conn *grpc.ClientConn,
	dotlog *dotlog.DotLog,
) ServerClientImpl {
	return &ServerClient{
		machineClient:      machine.NewMachineServiceClient(conn),
		loginSessionClient: login_session.NewLoginSessionServiceClient(conn),
		conn:               conn,
		ctx:                context.Background(),
		dotlog:             dotlog,
	}
}

// TODO: (shinta) remove SIGNAL_HOST and SIGNAL_PORT from env,
// use the SignalHost and SignalPort in response
func (c *ServerClient) GetMachine(mk, wgPubKey string) (*machine.GetMachineResponse, error) {
	md := metadata.New(map[string]string{utils.MachineKey: mk, utils.WgPubKey: wgPubKey})
	ctx := metadata.NewOutgoingContext(c.ctx, md)

	res, err := c.machineClient.GetMachine(ctx, &emptypb.Empty{})
	if err != nil {
		return nil, err
	}

	return &machine.GetMachineResponse{
		IsRegistered: res.IsRegistered,
		LoginUrl:     res.LoginUrl,
		Ip:           res.Ip,
		Cidr:         res.Cidr,
		SignalHost:   res.SignalHost,
		SignalPort:   res.SignalPort,
	}, nil
}

func (c *ServerClient) ConnectStreamPeerLoginSession(mk string) (*login_session.PeerLoginSessionResponse, error) {
	var (
		msg = &login_session.PeerLoginSessionResponse{}
	)

	sys := system.NewSysInfo()
	md := metadata.New(map[string]string{utils.MachineKey: mk, utils.HostName: sys.Hostname, utils.OS: sys.OS})
	newctx := metadata.NewOutgoingContext(c.ctx, md)

	stream, err := c.loginSessionClient.StreamPeerLoginSession(newctx, grpc.WaitForReady(true))
	if err != nil {
		return nil, err
	}

	header, err := stream.Header()
	if err != nil {
		return nil, err
	}

	sessionid := getLoginSessionID(header)
	c.dotlog.Logger.Debugf("sessionid: [%s]", sessionid)

	for {
		msg, err = stream.Recv()
		if err != nil {
			return nil, err
		}

		err = stream.Send(&emptypb.Empty{})
		if err != nil {
			return nil, err
		}
		break
	}

	return msg, nil
}

func (c *ServerClient) SyncRemoteMachinesConfig(mk string) (*machine.SyncMachinesResponse, error) {
	md := metadata.New(map[string]string{utils.MachineKey: mk})
	newctx := metadata.NewOutgoingContext(c.ctx, md)

	conf, err := c.machineClient.SyncRemoteMachinesConfig(newctx, &emptypb.Empty{})
	if err != nil {
		return nil, err
	}

	return conf, nil
}

// func (c *ServerClient) ConnectToHangoutMachines(mk string, handler func(msg *machine.HangOutMachinesResponse) error) error {
// 	md := metadata.New(map[string]string{utils.MachineKey: mk})
// 	newctx := metadata.NewOutgoingContext(c.ctx, md)

// 	stream, err := c.machineClient.ConnectToHangoutMachines(newctx, &emptypb.Empty{})
// 	if err != nil {
// 		return err
// 	}

// 	for {
// 		hangout, err := stream.Recv()
// 		if err == io.EOF {
// 			c.dotlog.Logger.Errorf("hangout machines return to EOF, received by [%s]", mk)
// 			return err
// 		}

// 		if err != nil {
// 			c.dotlog.Logger.Errorf("disconnect hangout machines, received by [%s], %s", mk, err.Error())
// 			return err
// 		}

// 		err = handler(hangout)
// 		if err != nil {
// 			c.dotlog.Logger.Errorf("error handle with hangout machines, received by [%s]", mk)
// 			return err
// 		}
// 	}
// }

func (c *ServerClient) JoinHangoutMachines(mk string) (*machine.HangOutMachinesResponse, error) {
	md := metadata.New(map[string]string{utils.MachineKey: mk})
	newctx := metadata.NewOutgoingContext(c.ctx, md)

	res, err := c.machineClient.JoinHangOutMachines(newctx, &emptypb.Empty{})
	if err != nil {
		return nil, err
	}

	return res, nil
}
