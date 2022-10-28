// Copyright (C) 2022 - Tillitis AB
// SPDX-License-Identifier: GPL-2.0-only

package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/spf13/pflag"
	"github.com/tillitis/tillitis-key1-apps/tk1"
)

var (
	cmdSetTimer     = appCmd{0x01, "cmdSetTimer", tk1.CmdLen32}
	rspSetTimer     = appCmd{0x02, "rspSetTimer", tk1.CmdLen4}
	cmdSetPrescaler = appCmd{0x03, "cmdSetPrescaler", tk1.CmdLen32}
	rspSetPrescaler = appCmd{0x04, "rspSetPrescaler", tk1.CmdLen4}
	cmdStartTimer   = appCmd{0x05, "cmdStartTimer", tk1.CmdLen1}
	rspStartTimer   = appCmd{0x06, "rspStartTimer", tk1.CmdLen4}
)

type appCmd struct {
	code   byte
	name   string
	cmdLen tk1.CmdLen
}

func (c appCmd) Code() byte {
	return c.code
}

func (c appCmd) CmdLen() tk1.CmdLen {
	return c.cmdLen
}

func (c appCmd) Endpoint() tk1.Endpoint {
	return tk1.DestApp
}

func (c appCmd) String() string {
	return c.name
}

type Timer struct {
	tk tk1.TillitisKey // A connection to a Tillitis Key 1
}

// New() gets you a connection to a timer app running on the Tillitis
// Key 1. You're expected to pass an existing TK1 connection to it, so
// use it like this:
//
//	tk, err := tk1.New(port, speed)
//	timer := NewTimer(tk)
func NewTimer(tk tk1.TillitisKey) Timer {
	var timer Timer

	timer.tk = tk

	return timer
}

// setInt sets an int with the command cmd
func (t Timer) setInt(sendCmd appCmd, expectedReceiveCmd appCmd, i int) error {
	id := 2
	tx, err := tk1.NewFrameBuf(sendCmd, id)
	if err != nil {
		return fmt.Errorf("NewFrameBuf: %w", err)
	}

	// The integer
	tx[2] = byte(i)
	tx[3] = byte(i >> 8)
	tx[4] = byte(i >> 16)
	tx[5] = byte(i >> 24)
	tk1.Dump("tx", tx)
	if err = t.tk.Write(tx); err != nil {
		return fmt.Errorf("Write: %w", err)
	}

	rx, _, err := t.tk.ReadFrame(expectedReceiveCmd, id)
	tk1.Dump("rx", rx)
	if err != nil {
		return fmt.Errorf("ReadFrame: %w", err)
	}

	if rx[2] != tk1.StatusOK {
		return fmt.Errorf("Command BAD")
	}

	return nil
}

func (t Timer) SetTimer(timer int) error {
	return t.setInt(cmdSetTimer, rspSetTimer, timer)
}

func (t Timer) SetPrescaler(prescaler int) error {
	return t.setInt(cmdSetPrescaler, rspSetPrescaler, prescaler)
}

func (t Timer) StartTimer() error {
	id := 2
	tx, err := tk1.NewFrameBuf(cmdStartTimer, id)
	if err != nil {
		return fmt.Errorf("NewFrameBuf: %w", err)
	}

	if err = t.tk.Write(tx); err != nil {
		return fmt.Errorf("Write: %w", err)
	}

	rx, _, err := t.tk.ReadFrame(rspStartTimer, id)
	tk1.Dump("rx", rx)
	if err != nil {
		return fmt.Errorf("ReadFrame: %w", err)
	}

	if rx[2] != tk1.StatusOK {
		return fmt.Errorf("Command BAD")
	}

	return nil
}

// matching device clock at 18 MHz
const defaultPrescaler = 18_000_000

func main() {
	port := pflag.String("port", "/dev/ttyACM0",
		"Set serial port device `PATH`.")
	speed := pflag.Int("speed", tk1.SerialSpeed,
		"Set serial port speed in `BPS` (bits per second).")
	verbose := pflag.Bool("verbose", false,
		"Enable verbose output.")
	timer := pflag.Int("timer", 1,
		fmt.Sprintf("Set timer `VALUE` (seconds if prescaler is %d).", defaultPrescaler))
	prescaler := pflag.Int("prescaler", defaultPrescaler,
		"Set prescaler.")
	pflag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage of %s:\n%s", os.Args[0],
			pflag.CommandLine.FlagUsagesWrapped(80))
	}
	pflag.Parse()

	if !*verbose {
		tk1.SilenceLogging()
	}

	fmt.Printf("Connecting to device on serial port %s ...\n", *port)
	tk, err := tk1.New(*port, *speed)
	if err != nil {
		fmt.Printf("Could not open %s: %v\n", *port, err)
		os.Exit(1)
	}
	exit := func(code int) {
		if err := tk.Close(); err != nil {
			fmt.Printf("tk.Close: %v\n", err)
		}
		os.Exit(code)
	}
	handleSignals(func() { exit(1) }, os.Interrupt, syscall.SIGTERM)

	tm := NewTimer(tk)

	err = tm.SetTimer(*timer)
	if err != nil {
		fmt.Print("SetTimer: %w", err)
		exit(1)
	}

	err = tm.SetPrescaler(*prescaler)
	if err != nil {
		fmt.Print("SetPrescaler: %w", err)
		exit(1)
	}

	t0 := time.Now()

	err = tm.StartTimer()
	if err != nil {
		fmt.Print("StartTimer: %w", err)
		exit(1)
	}

	elapsed := time.Since(t0)

	fmt.Printf("Timer expired after %v seconds\n", elapsed.Seconds())

	exit(0)
}

func handleSignals(action func(), sig ...os.Signal) {
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, sig...)
	go func() {
		for {
			<-ch
			action()
		}
	}()
}
