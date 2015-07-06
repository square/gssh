package erg

import (
	"bufio"
	"crypto/tls"
	"fmt"
	"github.com/square/gssh/Godeps/_workspace/src/github.com/xaviershay/grange"
	"net/http"
	"net/url"
	"sort"
)

// Erg type
// Sort boolean - turn it off/on for sorting on expand
// default is true
type Erg struct {
	host string
	port int
	ssl  bool
	Sort bool
}

// New(address string) returns a new erg
// takes two arguments
// host - hostname default - localhost
// port - port default - 8080
// ssl - use https or not default - false
func New(host string, port int) *Erg {
	return &Erg{host: host, port: port, ssl: false, Sort: true}
}

func NewWithSsl(host string, port int) *Erg {
	return &Erg{host: host, port: port, ssl: true, Sort: true}
}

// Expand takes a range expression as argument
// and returns an slice of strings as result
// err is set to nil on success
func (e *Erg) Expand(query string) (result []string, err error) {
	protocol := "http"

	if e.ssl {
		protocol = "https"
	}
	// TODO: Remove this with go 1.4
	// http://stackoverflow.com/questions/25008571/golang-issue-x509-cannot-verify-signature-algorithm-unimplemented-on-net-http
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{
			MaxVersion:               tls.VersionTLS11,
			PreferServerCipherSuites: true,
		},
	}
	client := &http.Client{Transport: tr}

	resp, err := client.Get(fmt.Sprintf("%s://%s:%d/range/list?%s",
		protocol,
		e.host,
		e.port,
		url.QueryEscape(query),
	))

	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()
	scanner := bufio.NewScanner(resp.Body)

	grangeResult := grange.NewResult()
	for scanner.Scan() {
		grangeResult.Add(scanner.Text())
	}

	if grangeResult.Cardinality() > 0 {
		for node := range grangeResult.Iter() {
			result = append(result, node.(string))
		}
		if e.Sort {
			sort.Strings(result)
		}
	}

	return result, nil
}

// Compress takes a slice of strings as argument
// and returns a compressed form.
func (*Erg) Compress(nodes []string) (result string) {
	grangeResult := grange.NewResult()
	for _, node := range nodes {
		grangeResult.Add(node)
	}
	return grange.Compress(&grangeResult)
}
