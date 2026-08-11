package main

import (
	"fmt"
	"os"

	"__TOOL__/internal/base"
	"__TOOL__/internal/version"
)

const usage = "usage: __TOOL__ [--version] [--help]"

func main() {
	switch base.Command(os.Args[1:]) {
	case "help":
		fmt.Println(usage)
	case "version":
		fmt.Println(version.Version)
	default:
		fmt.Println("__TOOL__")
	}
}
