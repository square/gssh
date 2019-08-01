Usage
======

```go
import "github.com/square/gcmd"

nodes = []string{"host1", "host2"}

// __NODE__ is replaced by each node
g := gcmd.New(nodes, "ssh", ,"__NODE__", "uname")

// maximum number of commands to run in parallel
g.Maxflight = 10
g.Run()

// you can override default stdout/stderr/exit
// handlers
g.StderrHandler = func(node string, o string) {
	fmt.Printf("%s:stderr:%s\n", node, string(o))
}
```
