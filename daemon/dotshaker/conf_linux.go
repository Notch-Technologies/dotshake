// Copyright (c) 2022 Notch Inc & AUTHORS All rights reserved.
// Use of this source code is governed by a BSD 3-Clause License
// license that can be found in the LICENSE file.

package daemon

var SystemConfig = `[Unit]
Description=dotshaker daemon
Requires=NetworkManager.service
After=network-online.target
Wants=network-online.target systemd-networkd-wait-online.service

[Service]
User=root
Type=simple
ExecStart=/usr/bin/dotshaker up -daemon=false
Restart=on-failure
RestartSec=15s

[Install]
WantedBy=multi-user.target
`

const DaemonFilePath = "/etc/systemd/system/dotshaker.service"
const BinPath = "/usr/bin/dotshaker"
const ServiceName = "dotshaker"
