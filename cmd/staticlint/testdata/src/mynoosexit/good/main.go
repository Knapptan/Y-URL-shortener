package main

import (
	"log"
)

// main использует log.Fatal вместо os.Exit - это корректный подход.
// Deferred функции будут выполнены корректно.
func main() {
	if true {
		log.Fatal("error occurred")
	}
}

//lint:file-ignore st1000 package comment not needed for test
