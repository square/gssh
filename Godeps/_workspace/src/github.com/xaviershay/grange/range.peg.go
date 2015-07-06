package grange

import (
	"fmt"
	"math"
	"sort"
	"strconv"
)

const end_symbol rune = 4

/* The rule types inferred from the grammar are below. */
type pegRule uint8

const (
	ruleUnknown pegRule = iota
	ruleexpression
	rulecombinedexpr
	rulerangeexpr
	rulecombinators
	ruleintersect
	ruleexclude
	ruleunion
	rulebraces
	rulebrackets
	rulegroupq
	rulecluster
	rulegroup
	rulekey
	rulelocalkey
	rulefunction
	rulefuncargs
	ruleregex
	ruleliteral
	rulevalue
	ruleleaderChar
	rulespace
	ruleconst
	ruleq
	rulequoted
	ruleAction0
	ruleAction1
	ruleAction2
	ruleAction3
	ruleAction4
	ruleAction5
	ruleAction6
	ruleAction7
	ruleAction8
	ruleAction9
	ruleAction10
	ruleAction11
	ruleAction12
	ruleAction13
	rulePegText
	ruleAction14
	ruleAction15
	ruleAction16
	ruleAction17

	rulePre_
	rule_In_
	rule_Suf
)

var rul3s = [...]string{
	"Unknown",
	"expression",
	"combinedexpr",
	"rangeexpr",
	"combinators",
	"intersect",
	"exclude",
	"union",
	"braces",
	"brackets",
	"groupq",
	"cluster",
	"group",
	"key",
	"localkey",
	"function",
	"funcargs",
	"regex",
	"literal",
	"value",
	"leaderChar",
	"space",
	"const",
	"q",
	"quoted",
	"Action0",
	"Action1",
	"Action2",
	"Action3",
	"Action4",
	"Action5",
	"Action6",
	"Action7",
	"Action8",
	"Action9",
	"Action10",
	"Action11",
	"Action12",
	"Action13",
	"PegText",
	"Action14",
	"Action15",
	"Action16",
	"Action17",

	"Pre_",
	"_In_",
	"_Suf",
}

type tokenTree interface {
	Print()
	PrintSyntax()
	PrintSyntaxTree(buffer string)
	Add(rule pegRule, begin, end, next, depth int)
	Expand(index int) tokenTree
	Tokens() <-chan token32
	AST() *node32
	Error() []token32
	trim(length int)
}

type node32 struct {
	token32
	up, next *node32
}

func (node *node32) print(depth int, buffer string) {
	for node != nil {
		for c := 0; c < depth; c++ {
			fmt.Printf(" ")
		}
		fmt.Printf("\x1B[34m%v\x1B[m %v\n", rul3s[node.pegRule], strconv.Quote(buffer[node.begin:node.end]))
		if node.up != nil {
			node.up.print(depth+1, buffer)
		}
		node = node.next
	}
}

func (ast *node32) Print(buffer string) {
	ast.print(0, buffer)
}

type element struct {
	node *node32
	down *element
}

/* ${@} bit structure for abstract syntax tree */
type token16 struct {
	pegRule
	begin, end, next int16
}

func (t *token16) isZero() bool {
	return t.pegRule == ruleUnknown && t.begin == 0 && t.end == 0 && t.next == 0
}

func (t *token16) isParentOf(u token16) bool {
	return t.begin <= u.begin && t.end >= u.end && t.next > u.next
}

func (t *token16) getToken32() token32 {
	return token32{pegRule: t.pegRule, begin: int32(t.begin), end: int32(t.end), next: int32(t.next)}
}

func (t *token16) String() string {
	return fmt.Sprintf("\x1B[34m%v\x1B[m %v %v %v", rul3s[t.pegRule], t.begin, t.end, t.next)
}

type tokens16 struct {
	tree    []token16
	ordered [][]token16
}

func (t *tokens16) trim(length int) {
	t.tree = t.tree[0:length]
}

func (t *tokens16) Print() {
	for _, token := range t.tree {
		fmt.Println(token.String())
	}
}

func (t *tokens16) Order() [][]token16 {
	if t.ordered != nil {
		return t.ordered
	}

	depths := make([]int16, 1, math.MaxInt16)
	for i, token := range t.tree {
		if token.pegRule == ruleUnknown {
			t.tree = t.tree[:i]
			break
		}
		depth := int(token.next)
		if length := len(depths); depth >= length {
			depths = depths[:depth+1]
		}
		depths[depth]++
	}
	depths = append(depths, 0)

	ordered, pool := make([][]token16, len(depths)), make([]token16, len(t.tree)+len(depths))
	for i, depth := range depths {
		depth++
		ordered[i], pool, depths[i] = pool[:depth], pool[depth:], 0
	}

	for i, token := range t.tree {
		depth := token.next
		token.next = int16(i)
		ordered[depth][depths[depth]] = token
		depths[depth]++
	}
	t.ordered = ordered
	return ordered
}

type state16 struct {
	token16
	depths []int16
	leaf   bool
}

func (t *tokens16) AST() *node32 {
	tokens := t.Tokens()
	stack := &element{node: &node32{token32: <-tokens}}
	for token := range tokens {
		if token.begin == token.end {
			continue
		}
		node := &node32{token32: token}
		for stack != nil && stack.node.begin >= token.begin && stack.node.end <= token.end {
			stack.node.next = node.up
			node.up = stack.node
			stack = stack.down
		}
		stack = &element{node: node, down: stack}
	}
	return stack.node
}

func (t *tokens16) PreOrder() (<-chan state16, [][]token16) {
	s, ordered := make(chan state16, 6), t.Order()
	go func() {
		var states [8]state16
		for i, _ := range states {
			states[i].depths = make([]int16, len(ordered))
		}
		depths, state, depth := make([]int16, len(ordered)), 0, 1
		write := func(t token16, leaf bool) {
			S := states[state]
			state, S.pegRule, S.begin, S.end, S.next, S.leaf = (state+1)%8, t.pegRule, t.begin, t.end, int16(depth), leaf
			copy(S.depths, depths)
			s <- S
		}

		states[state].token16 = ordered[0][0]
		depths[0]++
		state++
		a, b := ordered[depth-1][depths[depth-1]-1], ordered[depth][depths[depth]]
	depthFirstSearch:
		for {
			for {
				if i := depths[depth]; i > 0 {
					if c, j := ordered[depth][i-1], depths[depth-1]; a.isParentOf(c) &&
						(j < 2 || !ordered[depth-1][j-2].isParentOf(c)) {
						if c.end != b.begin {
							write(token16{pegRule: rule_In_, begin: c.end, end: b.begin}, true)
						}
						break
					}
				}

				if a.begin < b.begin {
					write(token16{pegRule: rulePre_, begin: a.begin, end: b.begin}, true)
				}
				break
			}

			next := depth + 1
			if c := ordered[next][depths[next]]; c.pegRule != ruleUnknown && b.isParentOf(c) {
				write(b, false)
				depths[depth]++
				depth, a, b = next, b, c
				continue
			}

			write(b, true)
			depths[depth]++
			c, parent := ordered[depth][depths[depth]], true
			for {
				if c.pegRule != ruleUnknown && a.isParentOf(c) {
					b = c
					continue depthFirstSearch
				} else if parent && b.end != a.end {
					write(token16{pegRule: rule_Suf, begin: b.end, end: a.end}, true)
				}

				depth--
				if depth > 0 {
					a, b, c = ordered[depth-1][depths[depth-1]-1], a, ordered[depth][depths[depth]]
					parent = a.isParentOf(b)
					continue
				}

				break depthFirstSearch
			}
		}

		close(s)
	}()
	return s, ordered
}

func (t *tokens16) PrintSyntax() {
	tokens, ordered := t.PreOrder()
	max := -1
	for token := range tokens {
		if !token.leaf {
			fmt.Printf("%v", token.begin)
			for i, leaf, depths := 0, int(token.next), token.depths; i < leaf; i++ {
				fmt.Printf(" \x1B[36m%v\x1B[m", rul3s[ordered[i][depths[i]-1].pegRule])
			}
			fmt.Printf(" \x1B[36m%v\x1B[m\n", rul3s[token.pegRule])
		} else if token.begin == token.end {
			fmt.Printf("%v", token.begin)
			for i, leaf, depths := 0, int(token.next), token.depths; i < leaf; i++ {
				fmt.Printf(" \x1B[31m%v\x1B[m", rul3s[ordered[i][depths[i]-1].pegRule])
			}
			fmt.Printf(" \x1B[31m%v\x1B[m\n", rul3s[token.pegRule])
		} else {
			for c, end := token.begin, token.end; c < end; c++ {
				if i := int(c); max+1 < i {
					for j := max; j < i; j++ {
						fmt.Printf("skip %v %v\n", j, token.String())
					}
					max = i
				} else if i := int(c); i <= max {
					for j := i; j <= max; j++ {
						fmt.Printf("dupe %v %v\n", j, token.String())
					}
				} else {
					max = int(c)
				}
				fmt.Printf("%v", c)
				for i, leaf, depths := 0, int(token.next), token.depths; i < leaf; i++ {
					fmt.Printf(" \x1B[34m%v\x1B[m", rul3s[ordered[i][depths[i]-1].pegRule])
				}
				fmt.Printf(" \x1B[34m%v\x1B[m\n", rul3s[token.pegRule])
			}
			fmt.Printf("\n")
		}
	}
}

func (t *tokens16) PrintSyntaxTree(buffer string) {
	tokens, _ := t.PreOrder()
	for token := range tokens {
		for c := 0; c < int(token.next); c++ {
			fmt.Printf(" ")
		}
		fmt.Printf("\x1B[34m%v\x1B[m %v\n", rul3s[token.pegRule], strconv.Quote(buffer[token.begin:token.end]))
	}
}

func (t *tokens16) Add(rule pegRule, begin, end, depth, index int) {
	t.tree[index] = token16{pegRule: rule, begin: int16(begin), end: int16(end), next: int16(depth)}
}

func (t *tokens16) Tokens() <-chan token32 {
	s := make(chan token32, 16)
	go func() {
		for _, v := range t.tree {
			s <- v.getToken32()
		}
		close(s)
	}()
	return s
}

func (t *tokens16) Error() []token32 {
	ordered := t.Order()
	length := len(ordered)
	tokens, length := make([]token32, length), length-1
	for i, _ := range tokens {
		o := ordered[length-i]
		if len(o) > 1 {
			tokens[i] = o[len(o)-2].getToken32()
		}
	}
	return tokens
}

/* ${@} bit structure for abstract syntax tree */
type token32 struct {
	pegRule
	begin, end, next int32
}

func (t *token32) isZero() bool {
	return t.pegRule == ruleUnknown && t.begin == 0 && t.end == 0 && t.next == 0
}

func (t *token32) isParentOf(u token32) bool {
	return t.begin <= u.begin && t.end >= u.end && t.next > u.next
}

func (t *token32) getToken32() token32 {
	return token32{pegRule: t.pegRule, begin: int32(t.begin), end: int32(t.end), next: int32(t.next)}
}

func (t *token32) String() string {
	return fmt.Sprintf("\x1B[34m%v\x1B[m %v %v %v", rul3s[t.pegRule], t.begin, t.end, t.next)
}

type tokens32 struct {
	tree    []token32
	ordered [][]token32
}

func (t *tokens32) trim(length int) {
	t.tree = t.tree[0:length]
}

func (t *tokens32) Print() {
	for _, token := range t.tree {
		fmt.Println(token.String())
	}
}

func (t *tokens32) Order() [][]token32 {
	if t.ordered != nil {
		return t.ordered
	}

	depths := make([]int32, 1, math.MaxInt16)
	for i, token := range t.tree {
		if token.pegRule == ruleUnknown {
			t.tree = t.tree[:i]
			break
		}
		depth := int(token.next)
		if length := len(depths); depth >= length {
			depths = depths[:depth+1]
		}
		depths[depth]++
	}
	depths = append(depths, 0)

	ordered, pool := make([][]token32, len(depths)), make([]token32, len(t.tree)+len(depths))
	for i, depth := range depths {
		depth++
		ordered[i], pool, depths[i] = pool[:depth], pool[depth:], 0
	}

	for i, token := range t.tree {
		depth := token.next
		token.next = int32(i)
		ordered[depth][depths[depth]] = token
		depths[depth]++
	}
	t.ordered = ordered
	return ordered
}

type state32 struct {
	token32
	depths []int32
	leaf   bool
}

func (t *tokens32) AST() *node32 {
	tokens := t.Tokens()
	stack := &element{node: &node32{token32: <-tokens}}
	for token := range tokens {
		if token.begin == token.end {
			continue
		}
		node := &node32{token32: token}
		for stack != nil && stack.node.begin >= token.begin && stack.node.end <= token.end {
			stack.node.next = node.up
			node.up = stack.node
			stack = stack.down
		}
		stack = &element{node: node, down: stack}
	}
	return stack.node
}

func (t *tokens32) PreOrder() (<-chan state32, [][]token32) {
	s, ordered := make(chan state32, 6), t.Order()
	go func() {
		var states [8]state32
		for i, _ := range states {
			states[i].depths = make([]int32, len(ordered))
		}
		depths, state, depth := make([]int32, len(ordered)), 0, 1
		write := func(t token32, leaf bool) {
			S := states[state]
			state, S.pegRule, S.begin, S.end, S.next, S.leaf = (state+1)%8, t.pegRule, t.begin, t.end, int32(depth), leaf
			copy(S.depths, depths)
			s <- S
		}

		states[state].token32 = ordered[0][0]
		depths[0]++
		state++
		a, b := ordered[depth-1][depths[depth-1]-1], ordered[depth][depths[depth]]
	depthFirstSearch:
		for {
			for {
				if i := depths[depth]; i > 0 {
					if c, j := ordered[depth][i-1], depths[depth-1]; a.isParentOf(c) &&
						(j < 2 || !ordered[depth-1][j-2].isParentOf(c)) {
						if c.end != b.begin {
							write(token32{pegRule: rule_In_, begin: c.end, end: b.begin}, true)
						}
						break
					}
				}

				if a.begin < b.begin {
					write(token32{pegRule: rulePre_, begin: a.begin, end: b.begin}, true)
				}
				break
			}

			next := depth + 1
			if c := ordered[next][depths[next]]; c.pegRule != ruleUnknown && b.isParentOf(c) {
				write(b, false)
				depths[depth]++
				depth, a, b = next, b, c
				continue
			}

			write(b, true)
			depths[depth]++
			c, parent := ordered[depth][depths[depth]], true
			for {
				if c.pegRule != ruleUnknown && a.isParentOf(c) {
					b = c
					continue depthFirstSearch
				} else if parent && b.end != a.end {
					write(token32{pegRule: rule_Suf, begin: b.end, end: a.end}, true)
				}

				depth--
				if depth > 0 {
					a, b, c = ordered[depth-1][depths[depth-1]-1], a, ordered[depth][depths[depth]]
					parent = a.isParentOf(b)
					continue
				}

				break depthFirstSearch
			}
		}

		close(s)
	}()
	return s, ordered
}

func (t *tokens32) PrintSyntax() {
	tokens, ordered := t.PreOrder()
	max := -1
	for token := range tokens {
		if !token.leaf {
			fmt.Printf("%v", token.begin)
			for i, leaf, depths := 0, int(token.next), token.depths; i < leaf; i++ {
				fmt.Printf(" \x1B[36m%v\x1B[m", rul3s[ordered[i][depths[i]-1].pegRule])
			}
			fmt.Printf(" \x1B[36m%v\x1B[m\n", rul3s[token.pegRule])
		} else if token.begin == token.end {
			fmt.Printf("%v", token.begin)
			for i, leaf, depths := 0, int(token.next), token.depths; i < leaf; i++ {
				fmt.Printf(" \x1B[31m%v\x1B[m", rul3s[ordered[i][depths[i]-1].pegRule])
			}
			fmt.Printf(" \x1B[31m%v\x1B[m\n", rul3s[token.pegRule])
		} else {
			for c, end := token.begin, token.end; c < end; c++ {
				if i := int(c); max+1 < i {
					for j := max; j < i; j++ {
						fmt.Printf("skip %v %v\n", j, token.String())
					}
					max = i
				} else if i := int(c); i <= max {
					for j := i; j <= max; j++ {
						fmt.Printf("dupe %v %v\n", j, token.String())
					}
				} else {
					max = int(c)
				}
				fmt.Printf("%v", c)
				for i, leaf, depths := 0, int(token.next), token.depths; i < leaf; i++ {
					fmt.Printf(" \x1B[34m%v\x1B[m", rul3s[ordered[i][depths[i]-1].pegRule])
				}
				fmt.Printf(" \x1B[34m%v\x1B[m\n", rul3s[token.pegRule])
			}
			fmt.Printf("\n")
		}
	}
}

func (t *tokens32) PrintSyntaxTree(buffer string) {
	tokens, _ := t.PreOrder()
	for token := range tokens {
		for c := 0; c < int(token.next); c++ {
			fmt.Printf(" ")
		}
		fmt.Printf("\x1B[34m%v\x1B[m %v\n", rul3s[token.pegRule], strconv.Quote(buffer[token.begin:token.end]))
	}
}

func (t *tokens32) Add(rule pegRule, begin, end, depth, index int) {
	t.tree[index] = token32{pegRule: rule, begin: int32(begin), end: int32(end), next: int32(depth)}
}

func (t *tokens32) Tokens() <-chan token32 {
	s := make(chan token32, 16)
	go func() {
		for _, v := range t.tree {
			s <- v.getToken32()
		}
		close(s)
	}()
	return s
}

func (t *tokens32) Error() []token32 {
	ordered := t.Order()
	length := len(ordered)
	tokens, length := make([]token32, length), length-1
	for i, _ := range tokens {
		o := ordered[length-i]
		if len(o) > 1 {
			tokens[i] = o[len(o)-2].getToken32()
		}
	}
	return tokens
}

func (t *tokens16) Expand(index int) tokenTree {
	tree := t.tree
	if index >= len(tree) {
		expanded := make([]token32, 2*len(tree))
		for i, v := range tree {
			expanded[i] = v.getToken32()
		}
		return &tokens32{tree: expanded}
	}
	return nil
}

func (t *tokens32) Expand(index int) tokenTree {
	tree := t.tree
	if index >= len(tree) {
		expanded := make([]token32, 2*len(tree))
		copy(expanded, tree)
		t.tree = expanded
	}
	return nil
}

type rangeQuery struct {
	currentLiteral string
	nodeStack      []parserNode

	Buffer string
	buffer []rune
	rules  [44]func() bool
	Parse  func(rule ...int) error
	Reset  func()
	tokenTree
}

type textPosition struct {
	line, symbol int
}

type textPositionMap map[int]textPosition

func translatePositions(buffer string, positions []int) textPositionMap {
	length, translations, j, line, symbol := len(positions), make(textPositionMap, len(positions)), 0, 1, 0
	sort.Ints(positions)

search:
	for i, c := range buffer[0:] {
		if c == '\n' {
			line, symbol = line+1, 0
		} else {
			symbol++
		}
		if i == positions[j] {
			translations[positions[j]] = textPosition{line, symbol}
			for j++; j < length; j++ {
				if i != positions[j] {
					continue search
				}
			}
			break search
		}
	}

	return translations
}

type parseError struct {
	p *rangeQuery
}

func (e *parseError) Error() string {
	tokens, error := e.p.tokenTree.Error(), "\n"
	positions, p := make([]int, 2*len(tokens)), 0
	for _, token := range tokens {
		positions[p], p = int(token.begin), p+1
		positions[p], p = int(token.end), p+1
	}
	translations := translatePositions(e.p.Buffer, positions)
	for _, token := range tokens {
		begin, end := int(token.begin), int(token.end)
		error += fmt.Sprintf("parse error near \x1B[34m%v\x1B[m (line %v symbol %v - line %v symbol %v):\n%v\n",
			rul3s[token.pegRule],
			translations[begin].line, translations[begin].symbol,
			translations[end].line, translations[end].symbol,
			/*strconv.Quote(*/ e.p.Buffer[begin:end] /*)*/)
	}

	return error
}

func (p *rangeQuery) PrintSyntaxTree() {
	p.tokenTree.PrintSyntaxTree(p.Buffer)
}

func (p *rangeQuery) Highlighter() {
	p.tokenTree.PrintSyntax()
}

func (p *rangeQuery) Execute() {
	buffer, begin, end := p.Buffer, 0, 0
	for token := range p.tokenTree.Tokens() {
		switch token.pegRule {
		case rulePegText:
			begin, end = int(token.begin), int(token.end)
		case ruleAction0:
			p.addBraceStart()
		case ruleAction1:
			p.addOperator(operatorIntersect)
		case ruleAction2:
			p.addOperator(operatorSubtract)
		case ruleAction3:
			p.addOperator(operatorUnion)
		case ruleAction4:
			p.addBraces()
		case ruleAction5:
			p.addGroupQuery()
		case ruleAction6:
			p.addValue(buffer[begin:end])
			p.addClusterLookup()
		case ruleAction7:
			p.addClusterLookup()
		case ruleAction8:
			p.addGroupLookup()
		case ruleAction9:
			p.addKeyLookup()
		case ruleAction10:
			p.addLocalClusterLookup(buffer[begin:end])
		case ruleAction11:
			p.addFunction(buffer[begin:end])
		case ruleAction12:
			p.addFuncArg()
		case ruleAction13:
			p.addFuncArg()
		case ruleAction14:
			p.addRegex(buffer[begin:end])
		case ruleAction15:
			p.addValue(buffer[begin:end])
		case ruleAction16:
			p.addConstant(buffer[begin:end])
		case ruleAction17:
			p.addConstant(buffer[begin:end])

		}
	}
}

func (p *rangeQuery) Init() {
	p.buffer = []rune(p.Buffer)
	if len(p.buffer) == 0 || p.buffer[len(p.buffer)-1] != end_symbol {
		p.buffer = append(p.buffer, end_symbol)
	}

	var tree tokenTree = &tokens16{tree: make([]token16, math.MaxInt16)}
	position, depth, tokenIndex, buffer, rules := 0, 0, 0, p.buffer, p.rules

	p.Parse = func(rule ...int) error {
		r := 1
		if len(rule) > 0 {
			r = rule[0]
		}
		matches := p.rules[r]()
		p.tokenTree = tree
		if matches {
			p.tokenTree.trim(tokenIndex)
			return nil
		}
		return &parseError{p}
	}

	p.Reset = func() {
		position, tokenIndex, depth = 0, 0, 0
	}

	add := func(rule pegRule, begin int) {
		if t := tree.Expand(tokenIndex); t != nil {
			tree = t
		}
		tree.Add(rule, begin, position, depth, tokenIndex)
		tokenIndex++
	}

	matchDot := func() bool {
		if buffer[position] != end_symbol {
			position++
			return true
		}
		return false
	}

	/*matchChar := func(c byte) bool {
		if buffer[position] == c {
			position++
			return true
		}
		return false
	}*/

	/*matchRange := func(lower byte, upper byte) bool {
		if c := buffer[position]; c >= lower && c <= upper {
			position++
			return true
		}
		return false
	}*/

	rules = [...]func() bool{
		nil,
		/* 0 expression <- <(combinedexpr? !.)> */
		func() bool {
			position0, tokenIndex0, depth0 := position, tokenIndex, depth
			{
				position1 := position
				depth++
				{
					position2, tokenIndex2, depth2 := position, tokenIndex, depth
					if !rules[rulecombinedexpr]() {
						goto l2
					}
					goto l3
				l2:
					position, tokenIndex, depth = position2, tokenIndex2, depth2
				}
			l3:
				{
					position4, tokenIndex4, depth4 := position, tokenIndex, depth
					if !matchDot() {
						goto l4
					}
					goto l0
				l4:
					position, tokenIndex, depth = position4, tokenIndex4, depth4
				}
				depth--
				add(ruleexpression, position1)
			}
			return true
		l0:
			position, tokenIndex, depth = position0, tokenIndex0, depth0
			return false
		},
		/* 1 combinedexpr <- <(rangeexpr combinators?)> */
		func() bool {
			position5, tokenIndex5, depth5 := position, tokenIndex, depth
			{
				position6 := position
				depth++
				if !rules[rulerangeexpr]() {
					goto l5
				}
				{
					position7, tokenIndex7, depth7 := position, tokenIndex, depth
					if !rules[rulecombinators]() {
						goto l7
					}
					goto l8
				l7:
					position, tokenIndex, depth = position7, tokenIndex7, depth7
				}
			l8:
				depth--
				add(rulecombinedexpr, position6)
			}
			return true
		l5:
			position, tokenIndex, depth = position5, tokenIndex5, depth5
			return false
		},
		/* 2 rangeexpr <- <(space (const / function / cluster / group / groupq / localkey / regex / value / brackets / (Action0 braces)) space)> */
		func() bool {
			position9, tokenIndex9, depth9 := position, tokenIndex, depth
			{
				position10 := position
				depth++
				if !rules[rulespace]() {
					goto l9
				}
				{
					position11, tokenIndex11, depth11 := position, tokenIndex, depth
					if !rules[ruleconst]() {
						goto l12
					}
					goto l11
				l12:
					position, tokenIndex, depth = position11, tokenIndex11, depth11
					if !rules[rulefunction]() {
						goto l13
					}
					goto l11
				l13:
					position, tokenIndex, depth = position11, tokenIndex11, depth11
					if !rules[rulecluster]() {
						goto l14
					}
					goto l11
				l14:
					position, tokenIndex, depth = position11, tokenIndex11, depth11
					if !rules[rulegroup]() {
						goto l15
					}
					goto l11
				l15:
					position, tokenIndex, depth = position11, tokenIndex11, depth11
					if !rules[rulegroupq]() {
						goto l16
					}
					goto l11
				l16:
					position, tokenIndex, depth = position11, tokenIndex11, depth11
					if !rules[rulelocalkey]() {
						goto l17
					}
					goto l11
				l17:
					position, tokenIndex, depth = position11, tokenIndex11, depth11
					if !rules[ruleregex]() {
						goto l18
					}
					goto l11
				l18:
					position, tokenIndex, depth = position11, tokenIndex11, depth11
					if !rules[rulevalue]() {
						goto l19
					}
					goto l11
				l19:
					position, tokenIndex, depth = position11, tokenIndex11, depth11
					if !rules[rulebrackets]() {
						goto l20
					}
					goto l11
				l20:
					position, tokenIndex, depth = position11, tokenIndex11, depth11
					if !rules[ruleAction0]() {
						goto l9
					}
					if !rules[rulebraces]() {
						goto l9
					}
				}
			l11:
				if !rules[rulespace]() {
					goto l9
				}
				depth--
				add(rulerangeexpr, position10)
			}
			return true
		l9:
			position, tokenIndex, depth = position9, tokenIndex9, depth9
			return false
		},
		/* 3 combinators <- <(space (union / intersect / exclude / braces))> */
		func() bool {
			position21, tokenIndex21, depth21 := position, tokenIndex, depth
			{
				position22 := position
				depth++
				if !rules[rulespace]() {
					goto l21
				}
				{
					position23, tokenIndex23, depth23 := position, tokenIndex, depth
					if !rules[ruleunion]() {
						goto l24
					}
					goto l23
				l24:
					position, tokenIndex, depth = position23, tokenIndex23, depth23
					if !rules[ruleintersect]() {
						goto l25
					}
					goto l23
				l25:
					position, tokenIndex, depth = position23, tokenIndex23, depth23
					if !rules[ruleexclude]() {
						goto l26
					}
					goto l23
				l26:
					position, tokenIndex, depth = position23, tokenIndex23, depth23
					if !rules[rulebraces]() {
						goto l21
					}
				}
			l23:
				depth--
				add(rulecombinators, position22)
			}
			return true
		l21:
			position, tokenIndex, depth = position21, tokenIndex21, depth21
			return false
		},
		/* 4 intersect <- <('&' rangeexpr Action1 combinators?)> */
		func() bool {
			position27, tokenIndex27, depth27 := position, tokenIndex, depth
			{
				position28 := position
				depth++
				if buffer[position] != rune('&') {
					goto l27
				}
				position++
				if !rules[rulerangeexpr]() {
					goto l27
				}
				if !rules[ruleAction1]() {
					goto l27
				}
				{
					position29, tokenIndex29, depth29 := position, tokenIndex, depth
					if !rules[rulecombinators]() {
						goto l29
					}
					goto l30
				l29:
					position, tokenIndex, depth = position29, tokenIndex29, depth29
				}
			l30:
				depth--
				add(ruleintersect, position28)
			}
			return true
		l27:
			position, tokenIndex, depth = position27, tokenIndex27, depth27
			return false
		},
		/* 5 exclude <- <('-' rangeexpr Action2 combinators?)> */
		func() bool {
			position31, tokenIndex31, depth31 := position, tokenIndex, depth
			{
				position32 := position
				depth++
				if buffer[position] != rune('-') {
					goto l31
				}
				position++
				if !rules[rulerangeexpr]() {
					goto l31
				}
				if !rules[ruleAction2]() {
					goto l31
				}
				{
					position33, tokenIndex33, depth33 := position, tokenIndex, depth
					if !rules[rulecombinators]() {
						goto l33
					}
					goto l34
				l33:
					position, tokenIndex, depth = position33, tokenIndex33, depth33
				}
			l34:
				depth--
				add(ruleexclude, position32)
			}
			return true
		l31:
			position, tokenIndex, depth = position31, tokenIndex31, depth31
			return false
		},
		/* 6 union <- <(',' rangeexpr Action3 combinators?)> */
		func() bool {
			position35, tokenIndex35, depth35 := position, tokenIndex, depth
			{
				position36 := position
				depth++
				if buffer[position] != rune(',') {
					goto l35
				}
				position++
				if !rules[rulerangeexpr]() {
					goto l35
				}
				if !rules[ruleAction3]() {
					goto l35
				}
				{
					position37, tokenIndex37, depth37 := position, tokenIndex, depth
					if !rules[rulecombinators]() {
						goto l37
					}
					goto l38
				l37:
					position, tokenIndex, depth = position37, tokenIndex37, depth37
				}
			l38:
				depth--
				add(ruleunion, position36)
			}
			return true
		l35:
			position, tokenIndex, depth = position35, tokenIndex35, depth35
			return false
		},
		/* 7 braces <- <('{' combinedexpr? '}' rangeexpr? Action4)> */
		func() bool {
			position39, tokenIndex39, depth39 := position, tokenIndex, depth
			{
				position40 := position
				depth++
				if buffer[position] != rune('{') {
					goto l39
				}
				position++
				{
					position41, tokenIndex41, depth41 := position, tokenIndex, depth
					if !rules[rulecombinedexpr]() {
						goto l41
					}
					goto l42
				l41:
					position, tokenIndex, depth = position41, tokenIndex41, depth41
				}
			l42:
				if buffer[position] != rune('}') {
					goto l39
				}
				position++
				{
					position43, tokenIndex43, depth43 := position, tokenIndex, depth
					if !rules[rulerangeexpr]() {
						goto l43
					}
					goto l44
				l43:
					position, tokenIndex, depth = position43, tokenIndex43, depth43
				}
			l44:
				if !rules[ruleAction4]() {
					goto l39
				}
				depth--
				add(rulebraces, position40)
			}
			return true
		l39:
			position, tokenIndex, depth = position39, tokenIndex39, depth39
			return false
		},
		/* 8 brackets <- <('(' combinedexpr? ')')> */
		func() bool {
			position45, tokenIndex45, depth45 := position, tokenIndex, depth
			{
				position46 := position
				depth++
				if buffer[position] != rune('(') {
					goto l45
				}
				position++
				{
					position47, tokenIndex47, depth47 := position, tokenIndex, depth
					if !rules[rulecombinedexpr]() {
						goto l47
					}
					goto l48
				l47:
					position, tokenIndex, depth = position47, tokenIndex47, depth47
				}
			l48:
				if buffer[position] != rune(')') {
					goto l45
				}
				position++
				depth--
				add(rulebrackets, position46)
			}
			return true
		l45:
			position, tokenIndex, depth = position45, tokenIndex45, depth45
			return false
		},
		/* 9 groupq <- <('?' rangeexpr Action5)> */
		func() bool {
			position49, tokenIndex49, depth49 := position, tokenIndex, depth
			{
				position50 := position
				depth++
				if buffer[position] != rune('?') {
					goto l49
				}
				position++
				if !rules[rulerangeexpr]() {
					goto l49
				}
				if !rules[ruleAction5]() {
					goto l49
				}
				depth--
				add(rulegroupq, position50)
			}
			return true
		l49:
			position, tokenIndex, depth = position49, tokenIndex49, depth49
			return false
		},
		/* 10 cluster <- <(('%' literal Action6 key?) / ('%' rangeexpr Action7 key?))> */
		func() bool {
			position51, tokenIndex51, depth51 := position, tokenIndex, depth
			{
				position52 := position
				depth++
				{
					position53, tokenIndex53, depth53 := position, tokenIndex, depth
					if buffer[position] != rune('%') {
						goto l54
					}
					position++
					if !rules[ruleliteral]() {
						goto l54
					}
					if !rules[ruleAction6]() {
						goto l54
					}
					{
						position55, tokenIndex55, depth55 := position, tokenIndex, depth
						if !rules[rulekey]() {
							goto l55
						}
						goto l56
					l55:
						position, tokenIndex, depth = position55, tokenIndex55, depth55
					}
				l56:
					goto l53
				l54:
					position, tokenIndex, depth = position53, tokenIndex53, depth53
					if buffer[position] != rune('%') {
						goto l51
					}
					position++
					if !rules[rulerangeexpr]() {
						goto l51
					}
					if !rules[ruleAction7]() {
						goto l51
					}
					{
						position57, tokenIndex57, depth57 := position, tokenIndex, depth
						if !rules[rulekey]() {
							goto l57
						}
						goto l58
					l57:
						position, tokenIndex, depth = position57, tokenIndex57, depth57
					}
				l58:
				}
			l53:
				depth--
				add(rulecluster, position52)
			}
			return true
		l51:
			position, tokenIndex, depth = position51, tokenIndex51, depth51
			return false
		},
		/* 11 group <- <('@' rangeexpr Action8)> */
		func() bool {
			position59, tokenIndex59, depth59 := position, tokenIndex, depth
			{
				position60 := position
				depth++
				if buffer[position] != rune('@') {
					goto l59
				}
				position++
				if !rules[rulerangeexpr]() {
					goto l59
				}
				if !rules[ruleAction8]() {
					goto l59
				}
				depth--
				add(rulegroup, position60)
			}
			return true
		l59:
			position, tokenIndex, depth = position59, tokenIndex59, depth59
			return false
		},
		/* 12 key <- <(':' rangeexpr Action9)> */
		func() bool {
			position61, tokenIndex61, depth61 := position, tokenIndex, depth
			{
				position62 := position
				depth++
				if buffer[position] != rune(':') {
					goto l61
				}
				position++
				if !rules[rulerangeexpr]() {
					goto l61
				}
				if !rules[ruleAction9]() {
					goto l61
				}
				depth--
				add(rulekey, position62)
			}
			return true
		l61:
			position, tokenIndex, depth = position61, tokenIndex61, depth61
			return false
		},
		/* 13 localkey <- <('$' literal Action10)> */
		func() bool {
			position63, tokenIndex63, depth63 := position, tokenIndex, depth
			{
				position64 := position
				depth++
				if buffer[position] != rune('$') {
					goto l63
				}
				position++
				if !rules[ruleliteral]() {
					goto l63
				}
				if !rules[ruleAction10]() {
					goto l63
				}
				depth--
				add(rulelocalkey, position64)
			}
			return true
		l63:
			position, tokenIndex, depth = position63, tokenIndex63, depth63
			return false
		},
		/* 14 function <- <(literal Action11 '(' funcargs ')')> */
		func() bool {
			position65, tokenIndex65, depth65 := position, tokenIndex, depth
			{
				position66 := position
				depth++
				if !rules[ruleliteral]() {
					goto l65
				}
				if !rules[ruleAction11]() {
					goto l65
				}
				if buffer[position] != rune('(') {
					goto l65
				}
				position++
				if !rules[rulefuncargs]() {
					goto l65
				}
				if buffer[position] != rune(')') {
					goto l65
				}
				position++
				depth--
				add(rulefunction, position66)
			}
			return true
		l65:
			position, tokenIndex, depth = position65, tokenIndex65, depth65
			return false
		},
		/* 15 funcargs <- <((combinedexpr? Action12 ';' funcargs) / (combinedexpr? Action13))> */
		func() bool {
			position67, tokenIndex67, depth67 := position, tokenIndex, depth
			{
				position68 := position
				depth++
				{
					position69, tokenIndex69, depth69 := position, tokenIndex, depth
					{
						position71, tokenIndex71, depth71 := position, tokenIndex, depth
						if !rules[rulecombinedexpr]() {
							goto l71
						}
						goto l72
					l71:
						position, tokenIndex, depth = position71, tokenIndex71, depth71
					}
				l72:
					if !rules[ruleAction12]() {
						goto l70
					}
					if buffer[position] != rune(';') {
						goto l70
					}
					position++
					if !rules[rulefuncargs]() {
						goto l70
					}
					goto l69
				l70:
					position, tokenIndex, depth = position69, tokenIndex69, depth69
					{
						position73, tokenIndex73, depth73 := position, tokenIndex, depth
						if !rules[rulecombinedexpr]() {
							goto l73
						}
						goto l74
					l73:
						position, tokenIndex, depth = position73, tokenIndex73, depth73
					}
				l74:
					if !rules[ruleAction13]() {
						goto l67
					}
				}
			l69:
				depth--
				add(rulefuncargs, position68)
			}
			return true
		l67:
			position, tokenIndex, depth = position67, tokenIndex67, depth67
			return false
		},
		/* 16 regex <- <('/' <(!'/' .)*> '/' Action14)> */
		func() bool {
			position75, tokenIndex75, depth75 := position, tokenIndex, depth
			{
				position76 := position
				depth++
				if buffer[position] != rune('/') {
					goto l75
				}
				position++
				{
					position77 := position
					depth++
				l78:
					{
						position79, tokenIndex79, depth79 := position, tokenIndex, depth
						{
							position80, tokenIndex80, depth80 := position, tokenIndex, depth
							if buffer[position] != rune('/') {
								goto l80
							}
							position++
							goto l79
						l80:
							position, tokenIndex, depth = position80, tokenIndex80, depth80
						}
						if !matchDot() {
							goto l79
						}
						goto l78
					l79:
						position, tokenIndex, depth = position79, tokenIndex79, depth79
					}
					depth--
					add(rulePegText, position77)
				}
				if buffer[position] != rune('/') {
					goto l75
				}
				position++
				if !rules[ruleAction14]() {
					goto l75
				}
				depth--
				add(ruleregex, position76)
			}
			return true
		l75:
			position, tokenIndex, depth = position75, tokenIndex75, depth75
			return false
		},
		/* 17 literal <- <<(leaderChar ([a-z] / [A-Z] / ([0-9] / [0-9]) / '-' / '_')*)>> */
		func() bool {
			position81, tokenIndex81, depth81 := position, tokenIndex, depth
			{
				position82 := position
				depth++
				{
					position83 := position
					depth++
					if !rules[ruleleaderChar]() {
						goto l81
					}
				l84:
					{
						position85, tokenIndex85, depth85 := position, tokenIndex, depth
						{
							position86, tokenIndex86, depth86 := position, tokenIndex, depth
							if c := buffer[position]; c < rune('a') || c > rune('z') {
								goto l87
							}
							position++
							goto l86
						l87:
							position, tokenIndex, depth = position86, tokenIndex86, depth86
							if c := buffer[position]; c < rune('A') || c > rune('Z') {
								goto l88
							}
							position++
							goto l86
						l88:
							position, tokenIndex, depth = position86, tokenIndex86, depth86
							{
								position90, tokenIndex90, depth90 := position, tokenIndex, depth
								if c := buffer[position]; c < rune('0') || c > rune('9') {
									goto l91
								}
								position++
								goto l90
							l91:
								position, tokenIndex, depth = position90, tokenIndex90, depth90
								if c := buffer[position]; c < rune('0') || c > rune('9') {
									goto l89
								}
								position++
							}
						l90:
							goto l86
						l89:
							position, tokenIndex, depth = position86, tokenIndex86, depth86
							if buffer[position] != rune('-') {
								goto l92
							}
							position++
							goto l86
						l92:
							position, tokenIndex, depth = position86, tokenIndex86, depth86
							if buffer[position] != rune('_') {
								goto l85
							}
							position++
						}
					l86:
						goto l84
					l85:
						position, tokenIndex, depth = position85, tokenIndex85, depth85
					}
					depth--
					add(rulePegText, position83)
				}
				depth--
				add(ruleliteral, position82)
			}
			return true
		l81:
			position, tokenIndex, depth = position81, tokenIndex81, depth81
			return false
		},
		/* 18 value <- <(<(leaderChar (':' / ([a-z] / [A-Z]) / ([0-9] / [0-9]) / '-' / '_' / '.')*)> Action15)> */
		func() bool {
			position93, tokenIndex93, depth93 := position, tokenIndex, depth
			{
				position94 := position
				depth++
				{
					position95 := position
					depth++
					if !rules[ruleleaderChar]() {
						goto l93
					}
				l96:
					{
						position97, tokenIndex97, depth97 := position, tokenIndex, depth
						{
							position98, tokenIndex98, depth98 := position, tokenIndex, depth
							if buffer[position] != rune(':') {
								goto l99
							}
							position++
							goto l98
						l99:
							position, tokenIndex, depth = position98, tokenIndex98, depth98
							{
								position101, tokenIndex101, depth101 := position, tokenIndex, depth
								if c := buffer[position]; c < rune('a') || c > rune('z') {
									goto l102
								}
								position++
								goto l101
							l102:
								position, tokenIndex, depth = position101, tokenIndex101, depth101
								if c := buffer[position]; c < rune('A') || c > rune('Z') {
									goto l100
								}
								position++
							}
						l101:
							goto l98
						l100:
							position, tokenIndex, depth = position98, tokenIndex98, depth98
							{
								position104, tokenIndex104, depth104 := position, tokenIndex, depth
								if c := buffer[position]; c < rune('0') || c > rune('9') {
									goto l105
								}
								position++
								goto l104
							l105:
								position, tokenIndex, depth = position104, tokenIndex104, depth104
								if c := buffer[position]; c < rune('0') || c > rune('9') {
									goto l103
								}
								position++
							}
						l104:
							goto l98
						l103:
							position, tokenIndex, depth = position98, tokenIndex98, depth98
							if buffer[position] != rune('-') {
								goto l106
							}
							position++
							goto l98
						l106:
							position, tokenIndex, depth = position98, tokenIndex98, depth98
							if buffer[position] != rune('_') {
								goto l107
							}
							position++
							goto l98
						l107:
							position, tokenIndex, depth = position98, tokenIndex98, depth98
							if buffer[position] != rune('.') {
								goto l97
							}
							position++
						}
					l98:
						goto l96
					l97:
						position, tokenIndex, depth = position97, tokenIndex97, depth97
					}
					depth--
					add(rulePegText, position95)
				}
				if !rules[ruleAction15]() {
					goto l93
				}
				depth--
				add(rulevalue, position94)
			}
			return true
		l93:
			position, tokenIndex, depth = position93, tokenIndex93, depth93
			return false
		},
		/* 19 leaderChar <- <([a-z] / [A-Z] / ([0-9] / [0-9]) / '.' / '_')> */
		func() bool {
			position108, tokenIndex108, depth108 := position, tokenIndex, depth
			{
				position109 := position
				depth++
				{
					position110, tokenIndex110, depth110 := position, tokenIndex, depth
					if c := buffer[position]; c < rune('a') || c > rune('z') {
						goto l111
					}
					position++
					goto l110
				l111:
					position, tokenIndex, depth = position110, tokenIndex110, depth110
					if c := buffer[position]; c < rune('A') || c > rune('Z') {
						goto l112
					}
					position++
					goto l110
				l112:
					position, tokenIndex, depth = position110, tokenIndex110, depth110
					{
						position114, tokenIndex114, depth114 := position, tokenIndex, depth
						if c := buffer[position]; c < rune('0') || c > rune('9') {
							goto l115
						}
						position++
						goto l114
					l115:
						position, tokenIndex, depth = position114, tokenIndex114, depth114
						if c := buffer[position]; c < rune('0') || c > rune('9') {
							goto l113
						}
						position++
					}
				l114:
					goto l110
				l113:
					position, tokenIndex, depth = position110, tokenIndex110, depth110
					if buffer[position] != rune('.') {
						goto l116
					}
					position++
					goto l110
				l116:
					position, tokenIndex, depth = position110, tokenIndex110, depth110
					if buffer[position] != rune('_') {
						goto l108
					}
					position++
				}
			l110:
				depth--
				add(ruleleaderChar, position109)
			}
			return true
		l108:
			position, tokenIndex, depth = position108, tokenIndex108, depth108
			return false
		},
		/* 20 space <- <' '*> */
		func() bool {
			{
				position118 := position
				depth++
			l119:
				{
					position120, tokenIndex120, depth120 := position, tokenIndex, depth
					if buffer[position] != rune(' ') {
						goto l120
					}
					position++
					goto l119
				l120:
					position, tokenIndex, depth = position120, tokenIndex120, depth120
				}
				depth--
				add(rulespace, position118)
			}
			return true
		},
		/* 21 const <- <(q / quoted)> */
		func() bool {
			position121, tokenIndex121, depth121 := position, tokenIndex, depth
			{
				position122 := position
				depth++
				{
					position123, tokenIndex123, depth123 := position, tokenIndex, depth
					if !rules[ruleq]() {
						goto l124
					}
					goto l123
				l124:
					position, tokenIndex, depth = position123, tokenIndex123, depth123
					if !rules[rulequoted]() {
						goto l121
					}
				}
			l123:
				depth--
				add(ruleconst, position122)
			}
			return true
		l121:
			position, tokenIndex, depth = position121, tokenIndex121, depth121
			return false
		},
		/* 22 q <- <('q' '(' <(!')' .)*> ')' Action16)> */
		func() bool {
			position125, tokenIndex125, depth125 := position, tokenIndex, depth
			{
				position126 := position
				depth++
				if buffer[position] != rune('q') {
					goto l125
				}
				position++
				if buffer[position] != rune('(') {
					goto l125
				}
				position++
				{
					position127 := position
					depth++
				l128:
					{
						position129, tokenIndex129, depth129 := position, tokenIndex, depth
						{
							position130, tokenIndex130, depth130 := position, tokenIndex, depth
							if buffer[position] != rune(')') {
								goto l130
							}
							position++
							goto l129
						l130:
							position, tokenIndex, depth = position130, tokenIndex130, depth130
						}
						if !matchDot() {
							goto l129
						}
						goto l128
					l129:
						position, tokenIndex, depth = position129, tokenIndex129, depth129
					}
					depth--
					add(rulePegText, position127)
				}
				if buffer[position] != rune(')') {
					goto l125
				}
				position++
				if !rules[ruleAction16]() {
					goto l125
				}
				depth--
				add(ruleq, position126)
			}
			return true
		l125:
			position, tokenIndex, depth = position125, tokenIndex125, depth125
			return false
		},
		/* 23 quoted <- <('"' <(!'"' .)*> '"' Action17)> */
		func() bool {
			position131, tokenIndex131, depth131 := position, tokenIndex, depth
			{
				position132 := position
				depth++
				if buffer[position] != rune('"') {
					goto l131
				}
				position++
				{
					position133 := position
					depth++
				l134:
					{
						position135, tokenIndex135, depth135 := position, tokenIndex, depth
						{
							position136, tokenIndex136, depth136 := position, tokenIndex, depth
							if buffer[position] != rune('"') {
								goto l136
							}
							position++
							goto l135
						l136:
							position, tokenIndex, depth = position136, tokenIndex136, depth136
						}
						if !matchDot() {
							goto l135
						}
						goto l134
					l135:
						position, tokenIndex, depth = position135, tokenIndex135, depth135
					}
					depth--
					add(rulePegText, position133)
				}
				if buffer[position] != rune('"') {
					goto l131
				}
				position++
				if !rules[ruleAction17]() {
					goto l131
				}
				depth--
				add(rulequoted, position132)
			}
			return true
		l131:
			position, tokenIndex, depth = position131, tokenIndex131, depth131
			return false
		},
		/* 25 Action0 <- <{ p.addBraceStart() }> */
		func() bool {
			{
				add(ruleAction0, position)
			}
			return true
		},
		/* 26 Action1 <- <{ p.addOperator(operatorIntersect) }> */
		func() bool {
			{
				add(ruleAction1, position)
			}
			return true
		},
		/* 27 Action2 <- <{ p.addOperator(operatorSubtract) }> */
		func() bool {
			{
				add(ruleAction2, position)
			}
			return true
		},
		/* 28 Action3 <- <{ p.addOperator(operatorUnion) }> */
		func() bool {
			{
				add(ruleAction3, position)
			}
			return true
		},
		/* 29 Action4 <- <{ p.addBraces() }> */
		func() bool {
			{
				add(ruleAction4, position)
			}
			return true
		},
		/* 30 Action5 <- <{ p.addGroupQuery() }> */
		func() bool {
			{
				add(ruleAction5, position)
			}
			return true
		},
		/* 31 Action6 <- <{ p.addValue(buffer[begin:end]); p.addClusterLookup() }> */
		func() bool {
			{
				add(ruleAction6, position)
			}
			return true
		},
		/* 32 Action7 <- <{ p.addClusterLookup() }> */
		func() bool {
			{
				add(ruleAction7, position)
			}
			return true
		},
		/* 33 Action8 <- <{ p.addGroupLookup() }> */
		func() bool {
			{
				add(ruleAction8, position)
			}
			return true
		},
		/* 34 Action9 <- <{ p.addKeyLookup() }> */
		func() bool {
			{
				add(ruleAction9, position)
			}
			return true
		},
		/* 35 Action10 <- <{ p.addLocalClusterLookup(buffer[begin:end]) }> */
		func() bool {
			{
				add(ruleAction10, position)
			}
			return true
		},
		/* 36 Action11 <- <{ p.addFunction(buffer[begin:end]) }> */
		func() bool {
			{
				add(ruleAction11, position)
			}
			return true
		},
		/* 37 Action12 <- <{ p.addFuncArg() }> */
		func() bool {
			{
				add(ruleAction12, position)
			}
			return true
		},
		/* 38 Action13 <- <{ p.addFuncArg() }> */
		func() bool {
			{
				add(ruleAction13, position)
			}
			return true
		},
		nil,
		/* 40 Action14 <- <{ p.addRegex(buffer[begin:end]) }> */
		func() bool {
			{
				add(ruleAction14, position)
			}
			return true
		},
		/* 41 Action15 <- <{ p.addValue(buffer[begin:end]) }> */
		func() bool {
			{
				add(ruleAction15, position)
			}
			return true
		},
		/* 42 Action16 <- <{ p.addConstant(buffer[begin:end]) }> */
		func() bool {
			{
				add(ruleAction16, position)
			}
			return true
		},
		/* 43 Action17 <- <{ p.addConstant(buffer[begin:end]) }> */
		func() bool {
			{
				add(ruleAction17, position)
			}
			return true
		},
	}
	p.rules = rules
}
