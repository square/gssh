package grange

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

type RangeSpec struct {
	path    string
	line    int
	expr    string
	results Result
}

func TestCompress(t *testing.T) {
	spec_dir := os.Getenv("RANGE_SPEC_PATH")
	if spec_dir == "" {
		// Skip compress tests
		fmt.Fprintln(os.Stderr, "Skipping Compress() tests, RANGE_SPEC_PATH not set.")
		return
	}

	filepath.Walk(spec_dir+"/spec/compress", func(path string, info os.FileInfo, err error) error {
		if !info.IsDir() {
			return nil
		}

		specs, err := filepath.Glob(path + "/*.spec")
		if err == nil && specs != nil {
			for _, spec := range specs {
				loadSpec(t, spec)
			}
		}
		return nil
	})
}

func runSpec(t *testing.T, spec RangeSpec) {
	actual := Compress(&spec.results)

	if actual != spec.expr {
		t.Errorf("failed %s:%d\n got: %s\nwant: %s",
			spec.path, spec.line, actual, spec.expr)
	}
}

func loadSpec(t *testing.T, specpath string) {
	file, _ := os.Open(specpath)
	scanner := bufio.NewScanner(file)
	currentSpec := RangeSpec{results: NewResult(), path: specpath}

	line := 0
	for scanner.Scan() {
		line++
		if strings.HasPrefix(strings.Trim(scanner.Text(), " "), "#") {
			continue
		} else if scanner.Text() == "" {
			runSpec(t, currentSpec)
			currentSpec = RangeSpec{results: NewResult(), path: specpath}
		} else {
			if currentSpec.expr == "" {
				currentSpec.expr = scanner.Text()
				currentSpec.line = line
			} else {
				currentSpec.results.Add(scanner.Text())
			}
		}
	}
	if currentSpec.expr != "" {
		runSpec(t, currentSpec)
	}
}
