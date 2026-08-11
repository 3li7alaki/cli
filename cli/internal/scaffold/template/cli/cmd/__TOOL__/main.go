package main

import (
	"fmt"
	"os"

	"__TOOL__/internal/version"
)

func main() {
	if len(os.Args) == 2 && (os.Args[1] == "--version" || os.Args[1] == "version") {
		fmt.Println(version.Version)
		return
	}
	fmt.Println("__TOOL__")
}
