package main

import (
	"fmt"
	"os"

	"__TOOL__/internal/version"
)

const usage = "usage: __TOOL__ [--version] [--help]"

func main() {
	if len(os.Args) == 2 {
		switch os.Args[1] {
		case "-h", "--help", "help":
			fmt.Println(usage)
			return
		case "-v", "--version", "version":
			fmt.Println(version.Version)
			return
		}
	}
	fmt.Println("__TOOL__")
}
