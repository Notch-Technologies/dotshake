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

type ClientConf struct {
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

func NewClientConf(
	path string,
	serverHost string, serverPort uint,
	signalHost string, signalPort uint,
	isDebug bool,
	dl *dotlog.DotLog,
) (*ClientConf, error) {
	return &ClientConf{
		ServerHost: serverHost,
		ServerPort: serverPort,
		SignalHost: signalHost,
		SignalPort: signalPort,
		path:       path,
		isDebug:    isDebug,
		dotlog:     dl,
	}, nil
}

func (c *ClientConf) writeClientConf(
	wgPrivateKey, tunName string,
	serverHost string,
	serverPort uint,
	signalHost string,
	signalPort uint,
	blackList []string,
	presharedKey string,
) *ClientConf {
	if err := os.MkdirAll(filepath.Dir(c.path), 0755); err != nil {
		c.dotlog.Logger.Fatalf("failed to create directory with %s. because %s", c.path, err.Error())
	}

	c.ServerHost = serverHost
	c.ServerPort = serverPort
	c.SignalHost = signalHost
	c.SignalPort = signalPort
	c.WgPrivateKey = wgPrivateKey
	c.TunName = tunName
	c.BlackList = blackList

	b, err := json.MarshalIndent(*c, "", "\t")
	if err != nil {
		panic(err)
	}

	if err = utils.AtomicWriteFile(c.path, b, 0755); err != nil {
		panic(err)
	}

	return c
}

func (c *ClientConf) CreateClientConf() *ClientConf {
	b, err := ioutil.ReadFile(c.path)
	switch {
	case errors.Is(err, os.ErrNotExist):
		privKey, err := key.NewGenerateKey()
		if err != nil {
			c.dotlog.Logger.Error("failed to generate key for wireguard")
			panic(err)
		}

		return c.writeClientConf(
			privKey,
			tun.TunName(),
			c.ServerHost,
			c.ServerPort,
			c.SignalHost,
			c.SignalPort,
			[]string{tun.TunName()},
			"",
		)
	case err != nil:
		c.dotlog.Logger.Errorf("%s could not be read. exception error: %s", c.path, err.Error())
		panic(err)
	default:
		var core ClientConf
		if err := json.Unmarshal(b, &core); err != nil {
			c.dotlog.Logger.Fatalf("can not read client config file. because %v", err)
		}

		var serverhost string
		var signalhost string

		// TODO: (shinta) refactor
		// for daemon
		if c.ServerHost == "" {
			serverhost = core.ServerHost
		} else {
			serverhost = c.ServerHost
		}

		if c.SignalHost == "" {
			signalhost = core.SignalHost
		} else {
			signalhost = c.SignalHost
		}

		return c.writeClientConf(
			core.WgPrivateKey,
			core.TunName,
			serverhost,
			c.ServerPort,
			signalhost,
			c.SignalPort,
			core.BlackList,
			"",
		)
	}
}

func (c *ClientConf) GetClientConf() (*ClientConf, error) {
	var cc ClientConf
	b, err := ioutil.ReadFile(c.path)
	if err != nil {
		return nil, err
	}

	if err := json.Unmarshal(b, &cc); err != nil {
		c.dotlog.Logger.Fatalf("can not read client config file. because %v", err)
		return nil, err
	}

	cc.dotlog = c.dotlog

	return &cc, nil
}

// format like this => 127.0.0.1:443, ctl.dotshake.com:443
//
func (c *ClientConf) GetServerHost() string {
	return c.buildHost(c.ServerHost, c.ServerPort)
}

// format like this => 127.0.0.1:443, signal.dotshake.com:443
//
func (c *ClientConf) GetSignalHost() string {
	return c.buildHost(c.SignalHost, c.SignalPort)
}

func (c *ClientConf) buildHost(host string, port uint) string {
	var h string
	var p string
	if !c.isDebug {
		h = strings.Replace(host, "https://", "", -1)
	} else {
		h = strings.Replace(host, "http://", "", -1)
	}

	p = strconv.Itoa(int(port))
	return h + ":" + p
}
