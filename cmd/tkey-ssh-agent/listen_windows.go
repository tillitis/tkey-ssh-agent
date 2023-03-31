//go:build windows

package main

import (
	"fmt"
	"net"
	"os"
	"os/user"

	"github.com/Microsoft/go-winio"
)

func nativeListen(path string) (net.Listener, error) {
	if err := os.RemoveAll(path); err != nil {
		return nil, fmt.Errorf("RemoveAll: %w", err)
	}

	// Get the current user
	currentUser, err := user.Current()
	if err != nil {
		return nil, fmt.Errorf("CurrentUser: %w", err)
	}
	// TODO examine this:
	pc := &winio.PipeConfig{
		SecurityDescriptor: "D:(A;;FA;;;" + currentUser.Uid + ")",
		InputBufferSize:    4096,
		OutputBufferSize:   4096,
	}

	l, err := winio.ListenPipe(path, pc)
	if err != nil {
		return nil, fmt.Errorf("ListenPipe: %w", err)
	}
	return l, nil
}
