package main

import (
	"os"
)

// main вызывает os.Exit - это нарушение правила анализатора noosexit.
func main() {
	if true {
		os.Exit(1) // want `direct call to os.Exit in main function is prohibited`
	}
}

//lint:file-ignore st1000 package comment not needed for test
