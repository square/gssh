package grange

import (
	"fmt"
	"strings"
)

type operatorType int

const (
	operatorIntersect operatorType = iota
	operatorSubtract
	operatorUnion
)

type parserNode interface {
	String() string
}

type nodeNull struct{}

// Transient marker node to delineate the start of a braces capture. This is
// kind of weird. This node should never be present one parsing is complete.
type nodeBraceStart struct{}

type nodeText struct {
	val string
}

type nodeConstant struct {
	val string
}

type nodeRegexp struct {
	val string
}

type nodeLocalClusterLookup struct {
	key string
}

type nodeGroupQuery struct {
	node parserNode
}

type nodeClusterLookup struct {
	node parserNode
	key  parserNode
}

type nodeOperator struct {
	op    operatorType
	left  parserNode
	right parserNode
}

type nodeBraces struct {
	node  parserNode
	left  parserNode
	right parserNode
}

type nodeFunction struct {
	name   string
	params []parserNode
}

func (n nodeFunction) String() string {
	result := []string{}
	for _, param := range n.params {
		result = append(result, param.String())
	}

	return fmt.Sprintf("%s(%s)", n.name, strings.Join(result, ";"))
}

func (n nodeText) String() string {
	return n.val
}

func (n nodeConstant) String() string {
	return n.val
}

func (n nodeRegexp) String() string {
	return fmt.Sprintf("/%s/", n.val)
}

func (n nodeClusterLookup) String() string {
	switch n.key.(type) {
	case nodeText:
		if n.key.(nodeText).val == "CLUSTER" {
			return fmt.Sprintf("%%{%s}", n.node)
		}
	}
	return fmt.Sprintf("%%{%s}:%s", n.node, n.key)
}

func (n nodeGroupQuery) String() string {
	return fmt.Sprintf("?%s", n.node)
}

func (n nodeLocalClusterLookup) String() string {
	return fmt.Sprintf("$%s", n.key)
}

func (n nodeBraces) String() string {
	return fmt.Sprintf("%s{%s}%s", n.node, n.left, n.right)
}

func (n nodeNull) String() string {
	return ""
}

func (n nodeBraceStart) String() string {
	return ""
}

func (n nodeOperator) String() string {
	var op string

	switch n.op {
	case operatorIntersect:
		op = "&"
	case operatorSubtract:
		op = "-"
	case operatorUnion:
		op = ","
	}
	return fmt.Sprintf("%s %s %s", n.left, op, n.right)
}

func (t operatorType) String() string {
	switch t {
	case operatorIntersect:
		return "&"
	case operatorSubtract:
		return "-"
	case operatorUnion:
		return ","
	default:
		panic("Unknown operatorType")
	}
}
