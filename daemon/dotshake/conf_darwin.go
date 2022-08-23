// Copyright (c) 2022 Notch Inc & AUTHORS All rights reserved.
// Use of this source code is governed by a BSD 3-Clause License
// license that can be found in the LICENSE file.

package daemon

const SystemConfig = `
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>

  <key>Label</key>
  <string>com.dotshake.dotshake</string>

  <key>ProgramArguments</key>
  <array>
    <string>/usr/local/bin/dotshake</string>
    <string>up</string>
  </array>

  <key>RunAtLoad</key><true/>
  <key>StartInterval</key>
  <integer>5</integer>

  <key>StandardErrorPath</key>
  <string>/usr/local/var/log/dotshake.err</string>
  <key>StandardOutPath</key>
  <string>/usr/local/var/log/dotshake.log</string>
</dict>
</plist>
`

const DaemonFilePath = "/Library/LaunchDaemons/com.dotshake.dotshake.plist"
const BinPath = "/usr/local/bin/dotshake"
const ServiceName = "com.dotshake.dotshake"
