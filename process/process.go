// Copyright (c) 2022 Notch Inc & AUTHORS All rights reserved.
// Use of this source code is governed by a BSD 3-Clause License
// license that can be found in the LICENSE file.

package process

import "github.com/Notch-Technologies/dotshake/dotlog"

type Process interface {
	GetDotShakerProcess() bool
}

func NewProcess(
	dotlog *dotlog.DotLog,
) Process {
	return newProcess(dotlog)
}
