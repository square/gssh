// Copyright (c) 2014 Square, Inc

package main

import (
	"bufio"
	"flag"
	"fmt"
	"github.com/syamp/gcmd"
	"log"
	"os"
)

func main() {
	// options
	var maxflight, timeout int
	var file string

	flag.IntVar(&maxflight, "m", 50,
		"maximum number of parallel processes, default - 50")
	flag.IntVar(&maxflight, "maxflight", 50,
		"maximum number of parallel processes, default - 50")
	flag.IntVar(&timeout, "t", 10, "timeout in seconds for initial conn, default - 10s")
	flag.IntVar(&timeout, "timeout", 10,
		"timeout in seconds for initial conn, default - 10s")
	flag.StringVar(&file, "f", "",
		"file to read hostnames from default - stdin")
	flag.StringVar(&file, "file", "",
		"file to read hostnames from default - stdin")
	flag.Parse()

	var nodes   []string
	var scanner *bufio.Scanner

	if file == "" {
		scanner = bufio.NewScanner(os.Stdin)
	} else {
		f, err := os.Open(file)
		if err != nil {
			log.Fatal("open:", file, err.Error())
		}
		scanner = bufio.NewScanner(f)

	}

	for scanner.Scan() {
		nodes = append(nodes, scanner.Text())
	}

	timeout_arg := fmt.Sprintf("ConnectTimeout=%d", timeout)

	args := []string{"__NODE__", "-n", "-o", timeout_arg} // marker
	args = append(args, flag.Args()...)
	g := gcmd.New(nodes, "ssh", args...)
	g.Maxflight = maxflight
	g.Run()
}
