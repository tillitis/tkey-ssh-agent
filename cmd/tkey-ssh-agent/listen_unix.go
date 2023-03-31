//go:build linux || freebsd || openbsd || darwin

package main

import (
	"fmt"
	"net"
	"syscall"
)

func nativeListen(path string) (net.Listener, error) {
	syscall.Umask(0o077)
	l, err := net.Listen("unix", path)
	if err != nil {
		return nil, fmt.Errorf("Listen: %w", err)
	}
	return l, nil
}
