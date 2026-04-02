//go:build windows

package main

import (
	"fmt"
	"os"
	"os/exec"
)

func handleServiceCommand() (bool, error) {
	if len(os.Args) < 2 {
		return false, nil
	}
	exe, err := os.Executable()
	if err != nil {
		return true, err
	}
	switch os.Args[1] {
	case "install":
		cmd := exec.Command("sc.exe", "create", "WageSlaveMonitorAgent", "binPath=", fmt.Sprintf("\"%s\"", exe), "start=", "auto")
		return true, cmd.Run()
	case "uninstall":
		stop := exec.Command("sc.exe", "stop", "WageSlaveMonitorAgent")
		_ = stop.Run()
		cmd := exec.Command("sc.exe", "delete", "WageSlaveMonitorAgent")
		return true, cmd.Run()
	default:
		return false, nil
	}
}
