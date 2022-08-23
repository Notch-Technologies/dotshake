// Copyright (c) 2022 Notch Inc & AUTHORS All rights reserved.
// Use of this source code is governed by a BSD 3-Clause License
// license that can be found in the LICENSE file.

package process

import (
	"os/exec"

	"github.com/Notch-Technologies/dotshake/dotlog"
)

type dotshakerProcessOnLinux struct {
	dotlog *dotlog.DotLog
}

func newProcess(
	dotlog *dotlog.DotLog,
) Process {
	return &dotshakerProcessOnLinux{
		dotlog: dotlog,
	}
}

func (d *dotshakerProcessOnLinux) GetDotShakerProcess() bool {
	cmd := exec.Command("pgrep", "dotshaker")
	if out, err := cmd.CombinedOutput(); err != nil {
		d.dotlog.Logger.Errorf("Command: %v failed with output %s and error: %v", cmd.String(), out, err)
		return false
	}
	return true
}
