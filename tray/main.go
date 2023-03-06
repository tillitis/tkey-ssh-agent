package main

import (
	"fmt"
	"os/exec"

	"github.com/getlantern/systray"
)

func main() {
	onExit := func() {
	}
	systray.Run(onReady, onExit)
}

func onReady() {
	systray.SetTemplateIcon(Data, Data)

	mQuitOrig := systray.AddMenuItem("Quit", "Quit the whole app")


	go func () {

	cmd := exec.Command("../tkey-ssh-agent.exe", "--uss", "-a", "agent.sock")
	err := cmd.Start()
	if err != nil {
		fmt.Println(err)
		return
	}

	fmt.Println("Process started with PID:", cmd.Process.Pid)

	<-mQuitOrig.ClickedCh
	err = cmd.Process.Kill()
	if err != nil {
		fmt.Println(err)
	}
	fmt.Println("Process killed with PID:", cmd.Process.Pid)
	fmt.Println("Requesting quit")
		systray.Quit()
		fmt.Println("Finished quitting")
	}()
}
