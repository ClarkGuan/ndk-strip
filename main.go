package main

import (
	"fmt"
	"log"
	"os"
)

func main() {
	if len(os.Args) <= 1 {
		printHelp()
		return
	}

	filePath := os.Args[len(os.Args)-1]
	a, err := arch(filePath)
	if err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		printHelp()
		return
	}
	path, err := stripPath(a)
	if err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		printHelp()
		return
	}
	_ = run(path, os.Args[1:]...)
}

func printHelp() {
	path, err := stripPath("arm")
	if err != nil {
		log.Fatalln(err)
	}
	_ = run(path, "--help")
}
