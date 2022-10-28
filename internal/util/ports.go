// Copyright (C) 2022 - Tillitis AB
// SPDX-License-Identifier: GPL-2.0-only

package util

import (
	"fmt"
	"os"

	"go.bug.st/serial/enumerator"
)

const (
	tillitisUSBVID = "1207"
	tillitisUSBPID = "8887"
)

type SerialPort struct {
	DevPath      string
	SerialNumber string
}

func DetectSerialPort() (string, error) {
	ports, err := GetSerialPorts()
	if err != nil {
		return "", err
	}
	if len(ports) == 0 {
		fmt.Fprintf(os.Stderr, "Could not detect any Tillitis Key serial ports. You may pass\n"+
			"a known path using the --port flag.\n")
		return "", nil
	}
	if len(ports) > 1 {
		fmt.Fprintf(os.Stderr, "Detected %d Tillitis Key serial ports:\n", len(ports))
		for _, p := range ports {
			fmt.Fprintf(os.Stderr, "%s with serial number %s\n", p.DevPath, p.SerialNumber)
		}
		fmt.Fprintf(os.Stderr, "Please choose one of the above by using the --port flag.\n")
		return "", nil
	}
	fmt.Fprintf(os.Stderr, "Auto-detected serial port %s\n", ports[0].DevPath)
	return ports[0].DevPath, nil
}

func GetSerialPorts() ([]SerialPort, error) {
	var ports []SerialPort
	portDetails, err := enumerator.GetDetailedPortsList()
	if err != nil {
		return nil, fmt.Errorf("GetDetailedPortsList: %w", err)
	}
	if len(portDetails) == 0 {
		return ports, nil
	}
	for _, port := range portDetails {
		if port.IsUSB && port.VID == tillitisUSBVID && port.PID == tillitisUSBPID {
			ports = append(ports, SerialPort{port.Name, port.SerialNumber})
		}
	}
	return ports, nil
}
