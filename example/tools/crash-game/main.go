package main

import "flag"

// 放到main函数初始化，避免init()函数在测试时被调用，导致冲突
//func init() {
//	flag.Parse()
//	GetTonAPIIns()
//}

func main() {
	flag.Parse()
	GetTonAPIIns()

	// jetton操作
	JettonOperation()
	// crash game操作
	CrashGameOperation()
}
