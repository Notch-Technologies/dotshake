// Copyright (c) 2022 Notch Inc & AUTHORS All rights reserved.
// Use of this source code is governed by a BSD 3-Clause License
// license that can be found in the LICENSE file.

package paths

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
)

// state file to manage the secret information of the server.
// do not disclose to the outside world.
func DefaultDotshakeClientStateFile() string {
	switch runtime.GOOS {
	case "freebsd", "openbsd":
		return "/var/db/dotshake/client.state"
	case "linux":
		return "/var/lib/dotshake/client.state"
	case "darwin":
		return "/Library/dotshake/client.state"
	default:
		return ""
	}
}

func DefaultClientConfigFile() string {
	return "/etc/dotshake/client.json"
}

func DefaultClientLogFile() string {
	return "/var/log/dotshake/client.log"
}

func DefaultDotShakerLogFile() string {
	return "var/log/dotshaker/client.log"
}

func MkStateDir(dirPath string) error {
	if err := os.MkdirAll(dirPath, 0700); err != nil {
		return err
	}

	return checkStateDirPermission(dirPath)
}

func checkStateDirPermission(dir string) error {
	const (
		perm = 700
	)

	if filepath.Base(dir) != "dotshake" {
		return nil
	}

	fi, err := os.Stat(dir)
	if err != nil {
		return err
	}

	if !fi.IsDir() {
		return fmt.Errorf("expected %q is a directory, but %v", dir, fi.Mode())
	}

	if fi.Mode().Perm() == perm {
		return nil
	}

	return os.Chmod(dir, perm)
}
