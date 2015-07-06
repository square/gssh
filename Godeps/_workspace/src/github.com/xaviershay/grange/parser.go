package grange

func (r *rangeQuery) popNode() parserNode {
	l := len(r.nodeStack)
	result := r.nodeStack[l-1]
	r.nodeStack = r.nodeStack[:l-1]
	return result
}

func (r *rangeQuery) pushNode(node parserNode) {
	r.nodeStack = append(r.nodeStack, node)
}

func (r *rangeQuery) addValue(val string) {
	r.pushNode(nodeText{val})
}

func (r *rangeQuery) addConstant(val string) {
	r.pushNode(nodeConstant{val})
}

func (r *rangeQuery) addNull() {
	r.pushNode(nodeNull{})
}

func (r *rangeQuery) addBraceStart() {
	r.pushNode(nodeBraceStart{})
}

func (r *rangeQuery) addFuncArg() {
	var funcNode parserNode

	paramNode := r.popNode()
	switch paramNode.(type) {
	case nodeFunction:
		// No arguments. This is kind of terrible, probably a better way to do
		// this.
		r.pushNode(paramNode)
	default:
		funcNode = r.nodeStack[len(r.nodeStack)-1]
		fn := funcNode.(nodeFunction)
		fn.params = append(fn.params, paramNode)
		r.nodeStack[len(r.nodeStack)-1] = fn
	}
}

func (r *rangeQuery) addBraces() {
	right := r.popNode()
	node := r.popNode()

	var left parserNode
	left = nodeNull{}

	// This is kind of bullshit but not sure a better way to do it yet
	switch node.(type) {
	case nodeBraceStart:
		node = nodeNull{}
	default:
		if len(r.nodeStack) > 0 {
			left = r.popNode()
			switch left.(type) {
			case nodeBraceStart:
				left = nodeNull{}
			}
		}
	}
	r.pushNode(nodeBraces{node, left, right})
}

func (r *rangeQuery) addGroupLookup() {
	exprNode := r.popNode()
	r.pushNode(nodeClusterLookup{nodeConstant{"GROUPS"}, exprNode})
}

func (r *rangeQuery) addGroupQuery() {
	exprNode := r.popNode()
	r.pushNode(nodeGroupQuery{exprNode})
}

func (r *rangeQuery) addLocalClusterLookup(key string) {
	r.pushNode(nodeLocalClusterLookup{key})
}

func (r *rangeQuery) addFunction(name string) {
	r.pushNode(nodeFunction{name, []parserNode{}})
}

func (r *rangeQuery) addClusterLookup() {
	exprNode := r.popNode()
	r.pushNode(nodeClusterLookup{exprNode, nodeConstant{"CLUSTER"}})
}

func (r *rangeQuery) addRegex(val string) {
	r.pushNode(nodeRegexp{val})
}

func (r *rangeQuery) addKeyLookup() {
	keyNode := r.popNode()
	// TODO: Error out if no lookup
	if len(r.nodeStack) > 0 {
		lookupNode := r.popNode()

		switch lookupNode.(type) {
		case nodeClusterLookup:
			n := lookupNode.(nodeClusterLookup)
			n.key = keyNode
			r.pushNode(n)
			// TODO: Error out if wrong node type
		}
	}
}

func (r *rangeQuery) addOperator(typ operatorType) {
	right := r.popNode()
	left := r.popNode()

	r.pushNode(nodeOperator{typ, left, right})
}
