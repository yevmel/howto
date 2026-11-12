package prompt

import "golang.org/x/sys/unix"

func macOSVersion() string {
	v, _ := unix.Sysctl("kern.osproductversion")
	return v
}
