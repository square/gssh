/*
Grange implements a modern subset of the range query language. It is an
expressive grammar for selecting information out of arbitrary, self-referential
metadata. It was developed for querying information about hosts across
datacenters.

Basics

A range query operates on a state containing clusters.

    state := grange.NewState()
    state.AddCluster("a", Cluster{
      CLUSTER: []string{"a", "b", "c"},
      TYPE:    []string{"letters"},
    })
    result, err = state.Query("%a")      // "a", "b", "c"
    result, err = state.Query("%a:KEYS") // "CLUSTER", "TYPE"
    result, err = state.Query("%a:TYPE") // "letters"

Range also allows for a default cluster (traditionally named GROUPS), that can
be accessed with some shortcut syntax, documented below.

Values can also be range expressions, so that clusters can be defined in terms
of each other ("self-referential").

    state := grange.NewState()
    state.AddCluster("down", Cluster{ CLUSTER: []string{"host1"})
    state.AddCluster("dc1",  Cluster{ CLUSTER: []string{"@dc1 - %down"})

    result, err := state.Query("%dc1")  // "host2"

For an example usage of this library, see
https://github.com/xaviershay/grange-server

Syntax

    host1         - value constant, returns itself.
    host1,host2   - union, concatenates both sides.
    host1..3      - numeric expansion.
    a{b,c}d       - brace expansion, works just like your shell.
    (a,b) & a     - returns intersection of boths sides.
    (a,b) - a     - returns left side minus right side.
    /abc/         - regex match using RE2 semantics. When used on the right
                    side of an operator, filters the left side values using the
                    regex.  When used by itself, matches all values in the
                    default cluster..
    %dc1          - cluster lookup, returns the values at CLUSTER key in "dc1"
                    cluster.
    %dc1:KEYS     - returns all available keys for a cluster.
    %dc1:SOMEKEY  - returns values at SOMEKEY key.
    %dc1:{A,B}    - returns values at both A and B key. Query inside braces can
                    be any range expression.
    @dc1          - key lookup in default cluster, equivalent to %GROUPS:dc1.
    $SOMEKEY      - Looks up values from SOMEKEY in the current cluster when
                    used as a cluster value. When used at top-level, the
                    default cluster is used.
    ?host1        - returns all keys in the default cluster that contain host1.
    clusters(h1)  - returns all clusters for which the h1 is present in the
                    CLUSTER key. Parameter can be any range expression.
    has(KEY;val)  - returns all clusters with SOMEKEY matching value.
    count(EXPR)   - returns the number of results returned by EXPR.
    allclusters() - returns the names of all clusters
    q(x://blah)   - quote a constant value, the parameter will be returned as
                    is and not evaluated as a range expression. Useful for
                    storing metadata in clusters.

All of the above can be combined to form highly expressive queries.

    %{has(DC;east) & has(TYPE;redis)}:DOWN
        - all down redis nodes in the east datacenter.

    has(TYPE;%{clusters(host1)}:TYPE)
        - all clusters with types matching the clusters of host1.

    %{clusters(/foo/)}:{DOC,OWNER}
        - OWNER and DOC values for all clusters on all hosts matching "foo".

Differences From Libcrange

A number of libcrange features have been deliberately omitted from grange,
either becase they are archaic features of the language, or they are
mis-aligned with the goals of this library.

    - ^ "admin" operator is not supported. Not a useful concept anymore.
    - # "hash" operator is not supported. Normal function calls are sufficient.
    - Uses RE2 regular expressions rather than PCRE. RE2 is not as fully
      featured, but guarantees that searches run in time linear in the size of
      the input.  Regexes should not be used often anyway: prefer explicit
      metadata.
    - Non-deterministic functions, in particular functions that make network
      calls. This library aims to provide fast query performance, which is much
      harder when dealing with non-determinism. Clients who wish to emulate
      this behaviour should either calculate function results upfront and
      import them into the state, or post-process results.

*/
package grange
