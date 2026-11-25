//go:build !unix

package runner

import "os"

func reclaimTerminal(*os.File) {}
