//go:build !windows
// +build !windows

package main

func ensureAdmin() error {
	return nil
}
