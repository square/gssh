// Copyright (c) 2014 Square, Inc

package main

import (
	"bufio"
	"flag"
	"fmt"
	"log"
	"os"
	"strconv"

	"github.com/square/gcmd"
	"github.com/xaviershay/erg"
)

func main() {
	// options
	var maxflight, timeout int
	var file, rangeexp string
	var collapse bool

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
	// TODO: should be able to use any grouping system
	// TODO: perhaps use a cfg file to determine grouping system
	flag.StringVar(&rangeexp, "r", "",
		"rangeexp to read nodes from")
	flag.BoolVar(&collapse, "c", false,
		"collapse similar output - needs -r - be careful about memory usage")
	flag.BoolVar(&collapse, "collapse", false,
		"collapse similar output - needs -r - be careful about memory usage")
	flag.Parse()

	var nodes []string
	var scanner *bufio.Scanner
	var e *erg.Erg

	// read list of nodes from grouping system
	// or file/stdin
	if rangeexp != "" || collapse {
		host := "range"
		port := 80

		if envHost := os.Getenv("RANGE_HOST"); len(envHost) > 0 {
			host = envHost
		}

		if envPort := os.Getenv("RANGE_PORT"); len(envPort) > 0 {
			x, err := strconv.Atoi(envPort)
			if err != nil {
				log.Fatal("Invalid port in RANGE_PORT: ", envPort)
			}
			port = x
		}

		e = erg.New(host, port)
		result, err := e.Expand(rangeexp)
		nodes = result

		if err != nil {
			log.Fatal("Unable to expand: ", rangeexp, ":", err.Error())
		}

	} else {
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
	}

	timeout_arg := fmt.Sprintf("ConnectTimeout=%d", timeout)
	args := []string{"__NODE__", "-n", "-o", timeout_arg} // marker
	args = append(args, flag.Args()...)
	g := gcmd.New(nodes, "ssh", args...)
	g.Maxflight = maxflight

	// collapse output if asked to
	collapseStdout := map[string][]string{}
	collapseStderr := map[string][]string{}
	collapseExit := map[string][]string{}

	if collapse {
		g.StdoutHandler = func(node string, o string) {
			_, ok := collapseStdout[o]
			if !ok {
				collapseStdout[o] = make([]string, 0)
			}
			collapseStdout[o] = append(collapseStdout[o], node)
		}

		g.StderrHandler = func(node string, o string) {
			_, ok := collapseStderr[o]
			if !ok {
				collapseStderr[o] = make([]string, 0)
			}
			collapseStderr[o] = append(collapseStderr[o], node)
		}
		g.ExitHandler = func(node string, exit error) {
			o := "success"
			if exit != nil {
				o = exit.Error()
			}
			_, ok := collapseExit[o]
			if !ok {
				collapseExit[o] = make([]string, 0)
			}
			collapseExit[o] = append(collapseExit[o], node)
		}
	}

	g.Run()

	if collapse {
		for o, nodeArr := range collapseStdout {
			fmt.Println(e.Compress(nodeArr), "STDOUT", o)
		}
		for o, nodeArr := range collapseStderr {
			fmt.Println(e.Compress(nodeArr), "STDERR", o)
		}
		for o, nodeArr := range collapseExit {
			fmt.Println(e.Compress(nodeArr), "STATUS", o)
		}
	}
}
