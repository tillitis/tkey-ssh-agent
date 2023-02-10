//go:build linux || freebsd || openbsd || darwin

package main

import (
	"fmt"
	"net"
)

func nativeListen(path string) (net.Listener, error) {
	l, err := net.Listen("unix", path)
	if err != nil {
		return nil, fmt.Errorf("Listen: %w", err)
	}
	return l, nil
}
