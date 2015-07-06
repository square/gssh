package grange

import (
	"bufio"
	"fmt"
	"gopkg.in/v1/yaml"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestExpand(t *testing.T) {
	spec_dir := os.Getenv("RANGE_SPEC_PATH")
	if spec_dir == "" {
		// Skip compress tests
		fmt.Fprintln(os.Stderr, "Skipping Expand() tests, RANGE_SPEC_PATH not set.")
		return
	}

	filepath.Walk(spec_dir+"/spec/expand", func(path string, info os.FileInfo, err error) error {
		if !info.IsDir() {
			return nil
		}

		specs, err := filepath.Glob(path + "/*.spec")
		if err == nil && specs != nil {
			for _, spec := range specs {
				loadExpandSpec(t, spec)
			}
		}
		return nil
	})
}

func runExpandSpec(t *testing.T, spec RangeSpec) {
	state := NewState()
	// Load YAML files
	yamls, err := filepath.Glob(path.Dir(spec.path) + "/*.yaml")
	if err != nil {
		fmt.Println(err)
		return
	}

	for _, yamlPath := range yamls {
		dat, err := ioutil.ReadFile(yamlPath)
		if err != nil {
			t.Errorf("Could not read: %s", yamlPath)
		}
		basename := path.Base(yamlPath)
		name := strings.TrimSuffix(basename, ".yaml")

		m := make(map[string]interface{})
		err = yaml.Unmarshal(dat, &m)
		if err != nil {
			t.Errorf("Invalid YAML: %s", yamlPath)
		}
		c := yamlToCluster(name, m)
		state.AddCluster(name, c)
	}

	actual, err := state.Query(spec.expr)

	if err != nil {
		t.Errorf("%s", err)
	} else if !reflect.DeepEqual(actual, spec.results) {
		t.Errorf("failed %s:%d\n got: %s\nwant: %s",
			spec.path, spec.line, actual, spec.results)
	}
}

func loadExpandSpec(t *testing.T, specpath string) {
	file, _ := os.Open(specpath)
	scanner := bufio.NewScanner(file)
	currentSpec := RangeSpec{results: NewResult(), path: specpath}

	line := 0
	for scanner.Scan() {
		line++
		if strings.HasPrefix(strings.Trim(scanner.Text(), " "), "#") {
			continue
		} else if scanner.Text() == "" {
			runExpandSpec(t, currentSpec)
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
		runExpandSpec(t, currentSpec)
	}
}

// Converts a generic YAML map to a cluster by extracting all the correctly
// typed strings and discarding invalid values.
func yamlToCluster(clusterName string, yaml map[string]interface{}) Cluster {
	c := Cluster{}

	for key, value := range yaml {
		switch value.(type) {
		case nil:
			c[key] = []string{}
		case string:
			c[key] = []string{value.(string)}
		case int:
			c[key] = []string{fmt.Sprintf("%d", value.(int))}
		case bool:
			c[key] = []string{fmt.Sprintf("%s", value.(bool))}
		case []interface{}:
			result := []string{}

			for _, x := range value.([]interface{}) {
				switch x.(type) {
				case string:
					result = append(result, fmt.Sprintf("%s", x))
				case int:
					result = append(result, fmt.Sprintf("%d", x))
				case bool:
					result = append(result, fmt.Sprintf("%s", x))
				default:
					// discard
				}
			}
			c[key] = result
		default:
			// discard
		}
	}
	return c
}
