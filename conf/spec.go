// Copyright (c) 2022 Notch Inc & AUTHORS All rights reserved.
// Use of this source code is governed by a BSD 3-Clause License
// license that can be found in the LICENSE file.

package conf

import (
	"encoding/json"
	"errors"
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/Notch-Technologies/dotshake/dotlog"
	"github.com/Notch-Technologies/dotshake/tun"
	"github.com/Notch-Technologies/dotshake/types/key"
	"github.com/Notch-Technologies/dotshake/utils"
)

// spec is a dotshake config json
// path's here => '/etc/dotshake/client.json'
//
type Spec struct {
	WgPrivateKey string   `json:"wg_private_key"`
	ServerHost   string   `json:"server_host"`
	ServerPort   uint     `json:"server_port"`
	SignalHost   string   `json:"signal_host"`
	SignalPort   uint     `json:"signal_port"`
	TunName      string   `json:"tun"`
	PreSharedKey string   `json:"preshared_key"`
	BlackList    []string `json:"blacklist"`

	path    string
	isDebug bool

	dotlog *dotlog.DotLog
}

func NewSpec(
	path string,
	serverHost string, serverPort uint,
	signalHost string, signalPort uint,
	isDebug bool,
	dl *dotlog.DotLog,
) (*Spec, error) {
	return &Spec{
		ServerHost: serverHost,
		ServerPort: serverPort,
		SignalHost: signalHost,
		SignalPort: signalPort,
		path:       path,
		isDebug:    isDebug,
		dotlog:     dl,
	}, nil
}

func (s *Spec) writeSpec(
	wgPrivateKey, tunName string,
	serverHost string,
	serverPort uint,
	signalHost string,
	signalPort uint,
	blackList []string,
	presharedKey string,
) *Spec {
	if err := os.MkdirAll(filepath.Dir(s.path), 0755); err != nil {
		s.dotlog.Logger.Warnf("failed to create directory with %s, because %s", s.path, err.Error())
	}

	s.ServerHost = serverHost
	s.ServerPort = serverPort
	s.SignalHost = signalHost
	s.SignalPort = signalPort
	s.WgPrivateKey = wgPrivateKey
	s.TunName = tunName
	s.BlackList = blackList

	b, err := json.MarshalIndent(*s, "", "\t")
	if err != nil {
		panic(err)
	}

	if err = utils.AtomicWriteFile(s.path, b, 0755); err != nil {
		panic(err)
	}

	return s
}

func (s *Spec) CreateSpec() *Spec {
	b, err := ioutil.ReadFile(s.path)
	switch {
	case errors.Is(err, os.ErrNotExist):
		privKey, err := key.NewGenerateKey()
		if err != nil {
			s.dotlog.Logger.Error("failed to generate key for wireguard")
			panic(err)
		}

		return s.writeSpec(
			privKey,
			tun.TunName(),
			s.ServerHost,
			s.ServerPort,
			s.SignalHost,
			s.SignalPort,
			[]string{tun.TunName()},
			"",
		)
	case err != nil:
		s.dotlog.Logger.Errorf("%s could not be read. exception error: %s", s.path, err.Error())
		panic(err)
	default:
		var spec Spec
		if err := json.Unmarshal(b, &spec); err != nil {
			s.dotlog.Logger.Warnf("can not read client config file, because %v", err)
		}

		var serverhost string
		var signalhost string

		// TODO: (shinta) refactor
		// for daemon
		if s.ServerHost == "" {
			serverhost = spec.ServerHost
		} else {
			serverhost = s.ServerHost
		}

		if s.SignalHost == "" {
			signalhost = spec.SignalHost
		} else {
			signalhost = s.SignalHost
		}

		return s.writeSpec(
			spec.WgPrivateKey,
			spec.TunName,
			serverhost,
			s.ServerPort,
			signalhost,
			s.SignalPort,
			spec.BlackList,
			"",
		)
	}
}

func (s *Spec) GetClientConf() (*Spec, error) {
	var spec Spec
	b, err := ioutil.ReadFile(s.path)
	if err != nil {
		return nil, err
	}

	if err := json.Unmarshal(b, &spec); err != nil {
		s.dotlog.Logger.Warnf("can not read client config file, because %v", err)
		return nil, err
	}

	s.dotlog = s.dotlog

	return &spec, nil
}

// format like this => 127.0.0.1:443, ctl.dotshake.com:443
//
func (s *Spec) GetServerHost() string {
	return s.buildHost(s.ServerHost, s.ServerPort)
}

// format like this => 127.0.0.1:443, signal.dotshake.com:443
//
func (s *Spec) GetSignalHost() string {
	return s.buildHost(s.SignalHost, s.SignalPort)
}

func (s *Spec) buildHost(host string, port uint) string {
	var h string
	var p string
	if !s.isDebug {
		h = strings.Replace(host, "https://", "", -1)
	} else {
		h = strings.Replace(host, "http://", "", -1)
	}

	p = strconv.Itoa(int(port))
	return h + ":" + p
}
