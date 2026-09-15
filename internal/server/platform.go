package server

import "runtime"

func platform() string { return runtime.GOOS }
