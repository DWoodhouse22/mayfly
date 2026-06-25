//go:build !windows

package tunnel

func setupWintun() (string, error) { return "", nil }
