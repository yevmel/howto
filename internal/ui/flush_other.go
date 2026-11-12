//go:build !darwin && !linux

package ui

func flushInput(int) error { return nil }
