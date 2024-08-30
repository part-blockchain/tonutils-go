package main

import (
	"flag"
)

func init() {
	flag.Parse()
	GetTonAPIIns()
}

func main() {
	// jetton操作
	JettonOperation()
	// crash game操作
	CrashGameOperation()
}
