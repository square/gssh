package grange

import (
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"testing"
)

func TestEmptyQuery(t *testing.T) {
	testEval(t, NewResult(), "", emptyState())
}

func TestDefaultCluster(t *testing.T) {
	testEval(t, NewResult("b", "c"), "%a", singleCluster("a", Cluster{
		"CLUSTER": []string{"b", "c"},
	}))
}

func TestExplicitCluster(t *testing.T) {
	testEval(t, NewResult("b", "c"), "%a:NODES", singleCluster("a", Cluster{
		"NODES": []string{"b", "c"},
	}))
}

func TestClusterKeys(t *testing.T) {
	testEval(t, NewResult("NODES"), "%a:KEYS", singleCluster("a", Cluster{
		"NODES": []string{"b", "c"},
	}))
}

func TestClusterKeysMulti(t *testing.T) {
	testEval(t, NewResult("a", "b"), "%a:{NODES,TYPE}", singleCluster("a", Cluster{
		"NODES": []string{"a"},
		"TYPE":  []string{"b"},
	}))
}

func TestClusterMissing(t *testing.T) {
	testEval(t, NewResult(), "%a", emptyState())
}

func TestClusterMissingKey(t *testing.T) {
	testEval(t, NewResult(), "%a:NODES", singleCluster("a", Cluster{}))
}

func TestErrorExplicitCluster(t *testing.T) {
	testError(t, "Invalid token in query: \"}\"", "%a:}")
}

func TestErrorClusterName(t *testing.T) {
	testError(t, "Invalid token in query: \"}\"", "%}")
}

func TestStartingDash(t *testing.T) {
	testError(t, "Could not parse query: -foo", "-foo")
}

func TestHas(t *testing.T) {
	testEval(t, NewResult("a", "b"), "has(TYPE;one)", multiCluster(map[string]Cluster{
		"a": Cluster{"TYPE": []string{"one", "two"}},
		"b": Cluster{"TYPE": []string{"two", "one"}},
		"c": Cluster{"TYPE": []string{"three"}},
	}))
}

func TestHasIntersect(t *testing.T) {
	testEval(t, NewResult("b"), "has(TYPE;one)&b", multiCluster(map[string]Cluster{
		"a": Cluster{"TYPE": []string{"one", "two"}},
		"b": Cluster{"TYPE": []string{"two", "one"}},
		"c": Cluster{"TYPE": []string{"three"}},
	}))

	testEval(t, NewResult("b"), "has(TYPE;two)&has(TYPE;three)", multiCluster(map[string]Cluster{
		"a": Cluster{"TYPE": []string{"one", "two"}},
		"b": Cluster{"TYPE": []string{"two", "one", "three"}},
		"c": Cluster{"TYPE": []string{"three"}},
	}))
}

func TestIntersectEasy(t *testing.T) {
	testEval(t, NewResult("a"), "a & a", emptyState())
	testEval(t, NewResult(), "a & b", emptyState())
}

func TestIntersectCluster(t *testing.T) {
	testEval(t, NewResult("c"), "%a:L&%a:R", singleCluster("a", Cluster{
		"L": []string{"b", "c"},
		"R": []string{"c", "d"},
	}))
}

/*
// TODO: Pending
func TestIntersectError(t *testing.T) {
	testError(t, "No left side provided for intersection", "&a")
}
*/

func TestUnionEasy(t *testing.T) {
	testEval(t, NewResult("a", "b"), "a,b", emptyState())
}

func TestBracesWithUnion(t *testing.T) {
	testEval(t, NewResult("a.c", "b.c"), "{a,b}.c", emptyState())
	testEval(t, NewResult("a.b", "a.c"), "a.{b,c}", emptyState())
	testEval(t, NewResult("a.b.d", "a.c.d"), "a.{b,c}.d", emptyState())
}

func TestClusterUnion(t *testing.T) {
	testEval(t, NewResult("c", "d"), "%a,%b", multiCluster(map[string]Cluster{
		"a": Cluster{"CLUSTER": []string{"c"}},
		"b": Cluster{"CLUSTER": []string{"d"}},
	}))
}

/*
// TODO: Pending
func TestNoExpandInClusterName(t *testing.T) {
	testError(t, "Invalid token in query: \"{\"", "%a-{b,c}")
}
*/

func TestSelfReferentialCluster(t *testing.T) {
	testEval(t, NewResult("b"), "%a", multiCluster(map[string]Cluster{
		"a": Cluster{"CLUSTER": []string{"$ALL"}, "ALL": []string{"b"}},
	}))
}

func TestSelfReferentialClusterExpression(t *testing.T) {
	testEval(t, NewResult("a", "c"), "%a", multiCluster(map[string]Cluster{
		"a": Cluster{
			"CLUSTER": []string{"$ALL - $DOWN"},
			"ALL":     []string{"a", "b", "c"},
			"DOWN":    []string{"b"},
		},
	}))
}

func TestGroups(t *testing.T) {
	testEval(t, NewResult("a", "b"), "@dc", singleGroup("dc", "a", "b"))
}

func TestGroupsExpand(t *testing.T) {
	testEval(t, NewResult("c"), "@a", multiGroup(Cluster{
		"a": []string{"$b"},
		"b": []string{"c"},
	}))
}

func TestClusterLookup(t *testing.T) {
	testEval(t, NewResult("a"), "%{has(TYPE;db)}", singleCluster("ignore", Cluster{
		"CLUSTER": []string{"a"},
		"TYPE":    []string{"db"},
	}))
}

func TestClusterLookupExplicitKey(t *testing.T) {
	testEval(t, NewResult("a"), "%{has(TYPE;db)}:NODES", singleCluster("ignore", Cluster{
		"NODES": []string{"a"},
		"TYPE":  []string{"db"},
	}))
}

func TestClusterLookupDedup(t *testing.T) {
	testEval(t, NewResult("one", "two"), "%{has(TYPE;one)}:TYPE", multiCluster(map[string]Cluster{
		"a": Cluster{"TYPE": []string{"one", "two"}},
		"b": Cluster{"TYPE": []string{"two", "one"}},
		"c": Cluster{"TYPE": []string{"three"}},
	}))
}

func TestGroupsIsCluster(t *testing.T) {
	testEval(t, NewResult("a"), "%GROUPS:KEYS", singleGroup("a"))
}

func TestMatchNoContext(t *testing.T) {
	testEval(t, NewResult("ab"), "/b/", singleGroup("b", "ab", "c"))
}

func TestMatchRegexp(t *testing.T) {
	testEval(t, NewResult("ab"), "/^.b/", singleGroup("b", "ab", "cab"))
}

func TestInvalidRegexp(t *testing.T) {
	testError2(t, "error parsing regexp: missing argument to repetition operator: `+`", "/+/", emptyState())
}

func TestMatchEasy(t *testing.T) {
	testEval(t, NewResult("ab", "ba", "abc"), "%cluster & /b/",
		singleCluster("cluster", Cluster{
			"CLUSTER": []string{"ab", "ba", "abc", "ccc"},
		}))
}

func TestMatchReverse(t *testing.T) {
	testEval(t, NewResult("ab", "ba", "abc"), "/b/ & @group",
		singleGroup("group", "ab", "ba", "abc", "ccc"))
}

func TestMatchWithSubtract(t *testing.T) {
	testEval(t, NewResult("ccc"), "%cluster - /b/",
		singleCluster("cluster", Cluster{
			"CLUSTER": []string{"ab", "ba", "abc", "ccc"},
		}))
}

func TestUnionSubtractLeftAssociative(t *testing.T) {
	testEval(t, NewResult("a", "b-a"), "a,b-a", emptyState())
	testEval(t, NewResult("b"), "a , b - a", emptyState())
}

func TestCombineWithBraces(t *testing.T) {
	testEval(t, NewResult("b"), "b - %{a}", emptyState())
}

func TestGroupLookupAndSubtractiong(t *testing.T) {
	testEval(t, NewResult("a"), "{a} - b", emptyState())
}

func TestInvalidLex(t *testing.T) {
	testError(t, "No closing / for match", "/")
}

func TestClusters(t *testing.T) {
	testEval(t, NewResult("a", "b"), "clusters(one)", multiCluster(map[string]Cluster{
		"a": Cluster{"CLUSTER": []string{"two", "one"}},
		"b": Cluster{"CLUSTER": []string{"$ALL"}, "ALL": []string{"one"}},
		"c": Cluster{"CLUSTER": []string{"three"}},
	}))
}

func TestPrimeCacheReturnsErrors(t *testing.T) {
	state := singleGroup("a", "(")
	errors := state.PrimeCache()

	if len(errors) == 1 {
		expected := "Could not parse query: ("
		actual := errors[0].Error()
		if actual != expected {
			t.Errorf("Different error returned.\n got: %s\nwant: %s",
				actual, expected)
		}
	} else {
		t.Errorf("Expected 1 error, got %d", len(errors))
	}

}

func TestCycle(t *testing.T) {
	testError2(t, "Query exceeded maximum recursion limit", "%a",
		multiCluster(map[string]Cluster{
			"a": Cluster{"CLUSTER": []string{"%a"}},
		}))
}

func TestClustersEasy(t *testing.T) {
	testEval(t, NewResult("a"), "clusters(one)", multiCluster(map[string]Cluster{
		"a": Cluster{"CLUSTER": []string{"two", "one"}},
	}))
}

func TestQ(t *testing.T) {
	testEval(t, NewResult("(/"), "q((/)", emptyState())
	testEval(t, NewResult("http://foo/bar?yeah"), "q(http://foo/bar?yeah)", emptyState())
}

func TestQueryGroups(t *testing.T) {
	testEval(t, NewResult("one", "two"), "?a", multiGroup(Cluster{
		"one":   []string{"a"},
		"two":   []string{"$one"},
		"three": []string{"b"},
	}))
}

func TestCount(t *testing.T) {
	testEval(t, NewResult("1"), "count(a)", emptyState())
	testEval(t, NewResult("2"), "count({a,b,a})", emptyState())
	// TODO: why does this not parse
	// testEval(t, NewResult("2"), "count(a,b,a)", emptyState())
}

func TestAllClusters(t *testing.T) {
	testEval(t, NewResult("a"), "allclusters()", singleCluster("a", Cluster{}))
}

func TestLengthError(t *testing.T) {
	longString := strings.Repeat("a", MaxQuerySize)
	testEval(t, NewResult(longString), longString, emptyState())
	testError2(t, fmt.Sprintf("Query is too long, max length is %d", MaxQuerySize), longString+"a", emptyState())
}

func TestFunctionError(t *testing.T) {
	testError2(t, "Wrong number of params for has: expected 2, got 0.", "has()", emptyState())
	testError2(t, "Wrong number of params for has: expected 2, got 1.", "has(x)", emptyState())
	testError2(t, "Wrong number of params for has: expected 2, got 3.", "has(x;y;z)", emptyState())

	testError2(t, "Wrong number of params for count: expected 1, got 0.", "count()", emptyState())
	testError2(t, "Wrong number of params for clusters: expected 1, got 0.", "clusters()", emptyState())
	testError2(t, "Wrong number of params for allclusters: expected 0, got 1.", "allclusters(x)", emptyState())

	testError2(t, "Unknown function: foobar", "foobar(x)", emptyState())
}

func TestMaxResults(t *testing.T) {
	result := make([]interface{}, MaxResults)
	for i := 1; i <= MaxResults; i++ {
		result[i-1] = strconv.Itoa(i)
	}

	testEval(t, NewResult(result...), "1..10000000", emptyState())
}

func TestMaxText(t *testing.T) {
	longString := strings.Repeat("a", MaxQuerySize+1)
	testError2(t, "Value would exceed max query size: aaaaaaaaaaaaaaaaaaaa...", "%a",
		singleCluster("a", Cluster{
			"CLUSTER": []string{longString},
		}))
}

func BenchmarkClusters(b *testing.B) {
	// setup fake state
	state := NewState()

	state.AddCluster("cluster", Cluster{
		"CLUSTER": []string{"$ALL"},
		"ALL":     []string{"mynode"},
	})
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		state.Query("clusters(mynode)")
	}
}

func BenchmarkHas(b *testing.B) {
	// setup fake state
	state := NewState()

	state.AddCluster("cluster", Cluster{
		"CLUSTER": []string{"mynode"},
		"TYPE":    []string{"redis"},
	})
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		state.Query("has(TYPE;redis)")
	}
}

func testError(t *testing.T, expected string, query string) {
	_, err := emptyState().Query(query)

	if err == nil {
		t.Errorf("Expected error but none returned")
	} else if err.Error() != expected {
		// TODO: Get error messages back
		//t.Errorf("Different error returned.\n got: %s\nwant: %s", err.Error(), expected)
	}
}

func testError2(t *testing.T, expected string, query string, state *State) {
	_, err := state.Query(query)

	if err == nil {
		t.Errorf("Expected error but none returned")
	} else if err.Error() != expected {
		t.Errorf("Different error returned.\n got: %s\nwant: %s", err.Error(), expected)
	}
}

func testEval(t *testing.T, expected Result, query string, state *State) {
	actual, err := state.Query(query)

	if err != nil {
		t.Errorf("%s Expected result, got error: %s", query, err)
	} else if !reflect.DeepEqual(actual, expected) {
		t.Errorf("EvalRange\n got: %v\nwant: %v", actual, expected)
	}
}

func singleCluster(name string, c Cluster) *State {
	state := NewState()
	state.clusters[name] = c
	return &state
}

func singleGroup(name string, members ...string) *State {
	return singleCluster("GROUPS", Cluster{
		name: members,
	})
}

func multiGroup(c Cluster) *State {
	return singleCluster("GROUPS", c)
}

func multiCluster(cs map[string]Cluster) *State {
	state := NewState()
	state.clusters = cs
	return &state
}

func emptyState() *State {
	state := NewState()
	return &state
}
