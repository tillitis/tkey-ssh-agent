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
	"github.com/tillitis/tillitis-key1-apps/mkdf"
)

var (
	cmdSetTimer     = appCmd{0x01, "cmdSetTimer", mkdf.CmdLen32}
	rspSetTimer     = appCmd{0x02, "rspSetTimer", mkdf.CmdLen4}
	cmdSetPrescaler = appCmd{0x03, "cmdSetPrescaler", mkdf.CmdLen32}
	rspSetPrescaler = appCmd{0x04, "rspSetPrescaler", mkdf.CmdLen4}
	cmdStartTimer   = appCmd{0x05, "cmdStartTimer", mkdf.CmdLen1}
	rspStartTimer   = appCmd{0x06, "rspStartTimer", mkdf.CmdLen4}
)

type appCmd struct {
	code   byte
	name   string
	cmdLen mkdf.CmdLen
}

func (c appCmd) Code() byte {
	return c.code
}

func (c appCmd) CmdLen() mkdf.CmdLen {
	return c.cmdLen
}

func (c appCmd) Endpoint() mkdf.Endpoint {
	return mkdf.DestApp
}

func (c appCmd) String() string {
	return c.name
}

type Timer struct {
	tk mkdf.TillitisKey // A connection to a Tillitis Key 1
}

// New() gets you a connection to a timer app running on the Tillitis
// Key 1. You're expected to pass an existing TK1 connection to it, so
// use it like this:
//
//	tk, err := mkdf.New(port, speed)
//	timer := NewTimer(tk)
func NewTimer(tk mkdf.TillitisKey) Timer {
	var timer Timer

	timer.tk = tk

	return timer
}

// setInt sets an int with the command cmd
func (t Timer) setInt(sendCmd appCmd, expectedReceiveCmd appCmd, i int) error {
	id := 2
	tx, err := mkdf.NewFrameBuf(sendCmd, id)
	if err != nil {
		return fmt.Errorf("NewFrameBuf: %w", err)
	}

	// The integer
	tx[2] = byte(i)
	tx[3] = byte(i >> 8)
	tx[4] = byte(i >> 16)
	tx[5] = byte(i >> 24)
	mkdf.Dump("tx", tx)
	if err = t.tk.Write(tx); err != nil {
		return fmt.Errorf("Write: %w", err)
	}

	rx, _, err := t.tk.ReadFrame(expectedReceiveCmd, id)
	mkdf.Dump("rx", rx)
	if err != nil {
		return fmt.Errorf("ReadFrame: %w", err)
	}

	if rx[2] != mkdf.StatusOK {
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
	tx, err := mkdf.NewFrameBuf(cmdStartTimer, id)
	if err != nil {
		return fmt.Errorf("NewFrameBuf: %w", err)
	}

	if err = t.tk.Write(tx); err != nil {
		return fmt.Errorf("Write: %w", err)
	}

	rx, _, err := t.tk.ReadFrame(rspStartTimer, id)
	mkdf.Dump("rx", rx)
	if err != nil {
		return fmt.Errorf("ReadFrame: %w", err)
	}

	if rx[2] != mkdf.StatusOK {
		return fmt.Errorf("Command BAD")
	}

	return nil
}

func main() {
	port := pflag.String("port", "/dev/ttyACM0", "Serial port path")
	speed := pflag.Int("speed", mkdf.SerialSpeed, "When talking over the serial port, bits per second")
	verbose := pflag.Bool("verbose", false, "Enable verbose output")
	timer := pflag.Int("timer", 1, "Timer (seconds if default prescaler)")
	// matching device clock at 18 MHz
	prescaler := pflag.Int("prescaler", 18_000_000, "Prescaler")

	pflag.Parse()

	if !*verbose {
		mkdf.SilenceLogging()
	}

	fmt.Printf("Connecting to device on serial port %s ...\n", *port)
	tk, err := mkdf.New(*port, *speed)
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
