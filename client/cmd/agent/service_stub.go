//go:build !windows

package main

func handleServiceCommand() (bool, error) {
	return false, nil
}
