// Copyright (c) 2022 Notch Inc & AUTHORS All rights reserved.
// Use of this source code is governed by a BSD 3-Clause License
// license that can be found in the LICENSE file.

package webrtc

// this package provides the functions needed for udp hole punching using webrtc
// dependent on signal client
//

import (
	"github.com/Notch-Technologies/dotshake/client/grpc"
	"github.com/Notch-Technologies/dotshake/dotlog"
	"github.com/pion/ice/v2"
)

type SigExecuter struct {
	signalClient grpc.SignalClientImpl
	dstmk        string
	srcmk        string

	dotlog *dotlog.DotLog
}

func NewSigExecuter(
	signalClient grpc.SignalClientImpl,
	dstmk string,
	srcmk string,
	dotlog *dotlog.DotLog,
) *SigExecuter {
	return &SigExecuter{
		signalClient: signalClient,
		dstmk:        dstmk,
		srcmk:        srcmk,

		dotlog: dotlog,
	}
}

func (s *SigExecuter) Candidate(
	candidate ice.Candidate,
) {
	if candidate != nil {
		go func() {
			err := s.signalClient.Candidate(s.dstmk, s.srcmk, candidate)
			if err != nil {
				s.dotlog.Logger.Errorf("failed to candidate against signal server, becasuse %s", err.Error())
				return
			}
		}()
	}
}

func (s *SigExecuter) Offer(
	uFlag string,
	pwd string,
) error {
	return s.signalClient.Offer(s.dstmk, s.srcmk, uFlag, pwd)
}

func (s *SigExecuter) Answer(
	uFlag string,
	pwd string,
) error {
	return s.signalClient.Answer(s.dstmk, s.srcmk, uFlag, pwd)
}
