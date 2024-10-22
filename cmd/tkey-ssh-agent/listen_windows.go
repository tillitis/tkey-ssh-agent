// Copyright (C) 2023 - Tillitis AB
// SPDX-License-Identifier: BSD-2-Clause

//go:build windows

package main

import (
	"fmt"
	"net"
	"os/user"

	"github.com/Microsoft/go-winio"
)

func nativeListen(path string) (net.Listener, error) {
	// Create a SecurityDescriptor that makes the named pipe created
	// by ListenPipe accessible only by the current user
	currentUser, err := user.Current()
	if err != nil {
		return nil, fmt.Errorf("user.Current: %w", err)
	}
	pipeConf := &winio.PipeConfig{
		SecurityDescriptor: "D:(A;;FA;;;" + currentUser.Uid + ")",
		InputBufferSize:    4096,
		OutputBufferSize:   4096,
	}

	l, err := winio.ListenPipe(path, pipeConf)
	if err != nil {
		return nil, fmt.Errorf("ListenPipe: %w", err)
	}
	return l, nil
}
