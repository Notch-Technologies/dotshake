// Copyright (c) 2022 Notch Inc & AUTHORS All rights reserved.
// Use of this source code is governed by a BSD 3-Clause License
// license that can be found in the LICENSE file.

package iface

import (
	"fmt"
	"net"
	"time"

	"github.com/Notch-Technologies/dotshake/dotlog"
	"github.com/Notch-Technologies/dotshake/wg"
	"golang.zx2c4.com/wireguard/conn"
	"golang.zx2c4.com/wireguard/device"
	"golang.zx2c4.com/wireguard/tun"
	"golang.zx2c4.com/wireguard/wgctrl"
	"golang.zx2c4.com/wireguard/wgctrl/wgtypes"
)

type Iface struct {
	// your wireguard interface name
	Tun string
	// your wireguard private key
	WgPrivateKey string
	// your ip
	IP string
	// your cidr range
	CIDR string

	dotlog *dotlog.DotLog
}

func NewIface(
	tun, wgPrivateKey, ip string,
	cidr string, dotlog *dotlog.DotLog,
) *Iface {
	return &Iface{
		Tun:          tun,
		WgPrivateKey: wgPrivateKey,
		IP:           ip,
		CIDR:         cidr,

		dotlog: dotlog,
	}
}

func (i *Iface) ConfigureToRemotePeer(
	remotePeerPubKey, remoteip string,
	endpoint *net.UDPAddr,
	keepAlive time.Duration,
	preSharedKey string,
) error {
	i.dotlog.Logger.Debugf(
		"configuring %s to remote peer [%s:%s], remote endpoint [%s:%d]",
		i.Tun, remotePeerPubKey, remoteip, endpoint.IP.String(), endpoint.Port,
	)

	_, ipNet, err := net.ParseCIDR(remoteip)
	if err != nil {
		i.dotlog.Logger.Errorf("failed to parse cidr")
		return err
	}

	i.dotlog.Logger.Debugf("allowed remote ip [%s]", ipNet.IP.String())

	parsedRemotePeerPubKey, err := wgtypes.ParseKey(remotePeerPubKey)
	if err != nil {
		i.dotlog.Logger.Errorf("failed to remote peer pub key")
		return err
	}

	var parsedPreSharedkey wgtypes.Key
	if preSharedKey != "" {
		parsedPreSharedkey, err = wgtypes.ParseKey(preSharedKey)
		if err != nil {
			i.dotlog.Logger.Errorf("failed to wg preshared key")
			return err
		}
	}

	peer := wgtypes.PeerConfig{
		PublicKey:                   parsedRemotePeerPubKey,
		ReplaceAllowedIPs:           true,
		AllowedIPs:                  []net.IPNet{*ipNet},
		PersistentKeepaliveInterval: &keepAlive,
		PresharedKey:                &parsedPreSharedkey,
		Endpoint:                    endpoint,
	}

	config := wgtypes.Config{
		Peers: []wgtypes.PeerConfig{peer},
	}

	err = i.configureDevice(config)
	if err != nil {
		i.dotlog.Logger.Errorf("failed to configure device")
		return err
	}

	return nil
}

func (i *Iface) configureDevice(config wgtypes.Config) error {
	wg, err := wgctrl.New()
	if err != nil {
		i.dotlog.Logger.Errorf("failed to wgctl")
		return err
	}
	defer wg.Close()

	_, err = wg.Device(i.Tun)
	if err != nil {
		i.dotlog.Logger.Errorf("failed to wgdevice [%s], %s", i.Tun, err.Error())
		return err
	}

	return wg.ConfigureDevice(i.Tun, config)
}

func (i *Iface) RemoveRemotePeer(iface string, remoteip, remotePeerPubKey string) error {
	i.dotlog.Logger.Debugf("delete %s on %s", remotePeerPubKey, i.Tun)

	peerKeyParsed, err := wgtypes.ParseKey(remotePeerPubKey)
	if err != nil {
		return err
	}

	peer := wgtypes.PeerConfig{
		Remove:    true,
		PublicKey: peerKeyParsed,
	}

	config := wgtypes.Config{
		Peers: []wgtypes.PeerConfig{peer},
	}

	return i.configureDevice(config)
}

func (i *Iface) CreateWithUserSpace(address string) error {
	tunIface, err := tun.CreateTUN(i.Tun, wg.DefaultMTU)
	if err != nil {
		return err
	}

	tunDevice := device.NewDevice(tunIface, conn.NewDefaultBind(), device.NewLogger(device.LogLevelSilent, "wissy: "))
	err = tunDevice.Up()
	if err != nil {
		return err
	}

	uapi, err := getUAPI(i.Tun)
	if err != nil {
		return err
	}

	go func() {
		for {
			conn, err := uapi.Accept()
			if err != nil {
				fmt.Printf("uapi accept failed with error: %v\n", err)
				continue
			}
			go tunDevice.IpcHandle(conn)
		}
	}()

	err = assignAddr(i.Tun, address)
	if err != nil {
		return err
	}

	return nil
}
