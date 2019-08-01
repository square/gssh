package gcmd

import (
	"bufio"
	"fmt"
	"os/exec"
	"strings"
	"sync"
)

type StdoutHandlerFunc func(node string, o string)
type StderrHandlerFunc func(node string, e string)
type ExitHandlerFunc func(node string, exit error)

type Gcmd struct {
	Maxflight     int
	StdoutHandler StdoutHandlerFunc
	StderrHandler StderrHandlerFunc
	ExitHandler   ExitHandlerFunc
	command       string
	command_args  []string
	nodes         []string
	remaining     int
}

func New(nodes []string, command string, command_args ...string) *Gcmd {
	g := new(Gcmd)
	g.nodes = nodes
	g.command = command
	g.command_args = command_args
	// default handler functions
	g.StdoutHandler = func(node string, o string) {
		fmt.Printf("%s:stdout:%s\n", node, o)
	}

	g.StderrHandler = func(node string, o string) {
		fmt.Printf("%s:stderr:%s\n", node, o)
	}

	g.ExitHandler = func(node string, exit error) {
		if exit != nil {
			fmt.Printf("%s:failed:%s\n", node, exit.Error())
			return
		}
		fmt.Printf("%s:success\n", node)
	}

	return g
}

// Run the command with maxflight number of parallel
// processes and marker __NODE__ replaced with node
// name
func (g *Gcmd) Run() {

	maxflightChan := make(chan string, g.Maxflight)
	var wg sync.WaitGroup

	for g.remaining = len(g.nodes); g.remaining > 0; g.remaining-- {
		node := g.nodes[len(g.nodes)-g.remaining]
		maxflightChan <- node
		command_args := g.replaceMarker(node)

		// run each process in a goroutine
		wg.Add(1)
		go func(node string) {
			defer wg.Done()
			defer func() {
				<-maxflightChan
			}()

			cmd := exec.Command(g.command, command_args...)

			// setup stdout pipe
			stdout, err := cmd.StdoutPipe()
			if err != nil {
				g.ExitHandler(node, err)
				return
			}

			// setup stderr pipe
			stderr, err := cmd.StderrPipe()
			if err != nil {
				g.ExitHandler(node, err)
				return
			}

			// run the command
			if err = cmd.Start(); err != nil {
				g.ExitHandler(node, err)
				return
			}

			// read from stdout/stderr and invoke
			// user supplied handlers
			stdoutScanner := bufio.NewScanner(stdout)
			stderrScanner := bufio.NewScanner(stderr)

			wg.Add(1)
			go func() {
				defer wg.Done()
				for stdoutScanner.Scan() {
					g.StdoutHandler(node, stdoutScanner.Text())
				}
			}()

			wg.Add(1)
			go func() {
				defer wg.Done()
				for stderrScanner.Scan() {
					g.StderrHandler(node, stderrScanner.Text())
				}
			}()

			err = cmd.Wait()
			g.ExitHandler(node, err)
		}(node)
	}
	wg.Wait()
}

// unexported methods

// TODO: make replace marker configurable
func (g *Gcmd) replaceMarker(node string) []string {
	var command_args []string
	for _, arg := range g.command_args {
		command_args = append(command_args,
			strings.Replace(arg, "__NODE__", node, -1))
	}
	return command_args
}
