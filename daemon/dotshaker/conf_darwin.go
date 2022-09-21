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
  <string>com.dotshake.dotshaker</string>

  <key>ProgramArguments</key>
  <array>
    <string>/usr/local/bin/dotshaker</string>
    <string>up</string>
    <string>-daemon=false</string>
  </array>

  <key>RunAtLoad</key>
  <true/>

  <key>KeepAlive</key>
  <true/>

  <key>StartInterval</key>
    <integer>5</integer>

  <key>StandardErrorPath</key>
  <string>/usr/local/var/log/dotshaker.err</string>
  <key>StandardOutPath</key>
  <string>/usr/local/var/log/dotshaker.log</string>

</dict>
</plist>
`

const DaemonFilePath = "/Library/LaunchDaemons/com.dotshake.dotshaker.plist"
const BinPath = "/usr/local/bin/dotshaker"
const ServiceName = "com.dotshake.dotshaker"
