package main

import (
	"fmt"
	goopt "github.com/droundy/goopt"
	"github.com/square/gssh/Godeps/_workspace/src/github.com/xaviershay/erg"
	"os"
	"strconv"
)

var port = goopt.Int([]string{"-p", "--port"}, 8080, "Port to connect to. Can also be set with RANGE_PORT environment variable.")
var host = goopt.String([]string{"-h", "--host"}, "localhost", "Host to connect to. Can also be set with RANGE_HOST environment variable.")
var ssl = goopt.Flag([]string{"-s", "--ssl"}, []string{"--no-ssl"},
	"Don't use SSL", "Use SSL. Can also be set with RANGE_SSL environment variable.")
var expand = goopt.Flag([]string{"-e", "--expand"}, []string{"--no-expand"},
	"Do not compress results", "Compress results (default)")

func main() {
	if envHost := os.Getenv("RANGE_HOST"); len(envHost) > 0 {
		*host = envHost
	}

	if envSsl := os.Getenv("RANGE_SSL"); len(envSsl) > 0 {
		*ssl = true
	}

	if envPort := os.Getenv("RANGE_PORT"); len(envPort) > 0 {
		x, err := strconv.Atoi(envPort)
		if err == nil {
			*port = x
		} else {
			fmt.Fprintf(os.Stderr, "Invalid port in RANGE_PORT: %s\n", envPort)
			os.Exit(1)
		}
	}
	goopt.Parse(nil)

	var query string
	switch len(goopt.Args) {
	case 1:
		query = goopt.Args[0]
	default:
		fmt.Fprintln(os.Stderr, goopt.Usage())
		os.Exit(1)
	}

	var e *erg.Erg

	if *ssl {
		e = erg.NewWithSsl(*host, *port)
	} else {
		e = erg.New(*host, *port)
	}

	result, err := e.Expand(query)
	if err != nil {
		fmt.Println("Error: ", err.Error())
		os.Exit(1)
	}

	if *expand {
		for _, node := range result {
			fmt.Println(node)
		}
	} else {
		fmt.Println(e.Compress(result))
	}
}
