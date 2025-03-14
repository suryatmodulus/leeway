package leeway_test

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestFixtureLoadWorkspace(t *testing.T) {
	runDUT()

	tests := []*CommandFixtureTest{
		{
			Name:              "single workspace packages",
			T:                 t,
			Args:              []string{"collect", "-w", "fixtures/nested-ws/wsa"},
			NoNestedWorkspace: true,
			ExitCode:          0,
			StdoutSub:         "pkg1:app",
		},
		{
			Name:              "workspace components",
			T:                 t,
			Args:              []string{"collect", "-w", "fixtures/nested-ws/wsa", "components"},
			NoNestedWorkspace: true,
			ExitCode:          0,
			StdoutSub:         "//\npkg0\npkg1",
		},
		{
			Name:              "ignore nested workspaces",
			T:                 t,
			Args:              []string{"collect", "-w", "fixtures/nested-ws", "components"},
			NoNestedWorkspace: true,
			ExitCode:          1,
			StderrSub:         "pkg0:app: package \\\"wsa/pkg0:app\\\" is unknown",
		},
		{
			Name: "environment manifest",
			T:    t,
			Args: []string{"describe", "-w", "fixtures/nested-ws/wsa", "environment-manifest"},
			Eval: func(t *testing.T, stdout, stderr string) {
				for _, k := range []string{"os", "arch", "foobar"} {
					if !strings.Contains(stdout, fmt.Sprintf("%s: ", k)) {
						t.Errorf("missing %s entry in environment manifest", k)
					}
				}
			},
			ExitCode: 0,
		},
	}

	for _, test := range tests {
		test.Run()
	}
}

func TestPackageDefinition(t *testing.T) {
	runDUT()

	type pkginfo struct {
		Metadata struct {
			Version string `json:"version"`
		} `json:"metadata"`
	}

	tests := []struct {
		Name    string
		Layouts []map[string]string
		Tester  []func(t *testing.T, loc string, state map[string]string) *CommandFixtureTest
	}{
		{
			Name: "def change changes version",
			Layouts: []map[string]string{
				{
					"WORKSPACE.yaml":  "",
					"pkg1/BUILD.yaml": "packages:\n- name: foo\n  type: generic\n  srcs:\n  - \"doesNotExist\"",
				},
				{
					"pkg1/BUILD.yaml": "packages:\n- name: foo\n  type: generic\n  srcs:\n  - \"alsoDoesNotExist\"",
				},
			},
			Tester: []func(t *testing.T, loc string, state map[string]string) *CommandFixtureTest{
				func(t *testing.T, loc string, state map[string]string) *CommandFixtureTest {
					return &CommandFixtureTest{
						T:    t,
						Args: []string{"describe", "-w", loc, "-o", "json", "pkg1:foo"},
						Eval: func(t *testing.T, stdout, stderr string) {
							var dest pkginfo
							err := json.Unmarshal([]byte(stdout), &dest)
							if err != nil {
								fmt.Println(stdout)
								t.Fatal(err)
							}
							state["v"] = dest.Metadata.Version
						},
					}
				},
				func(t *testing.T, loc string, state map[string]string) *CommandFixtureTest {
					return &CommandFixtureTest{
						T:    t,
						Args: []string{"describe", "-w", loc, "-o", "json", "pkg1:foo"},
						Eval: func(t *testing.T, stdout, stderr string) {
							var dest pkginfo
							err := json.Unmarshal([]byte(stdout), &dest)
							if err != nil {
								fmt.Println(stdout)
								t.Fatal(err)
							}
							if state["v"] == dest.Metadata.Version {
								t.Errorf("definition change did not change version")
							}
						},
					}
				},
			},
		},
		{
			Name: "comp change doesnt change version",
			Layouts: []map[string]string{
				{
					"WORKSPACE.yaml":  "",
					"pkg1/BUILD.yaml": "packages:\n- name: foo\n  type: generic\n  srcs:\n  - \"doesNotExist\"",
				},
				{
					"pkg1/BUILD.yaml": "const:\n  foobar: baz\npackages:\n- name: foo\n  type: generic\n  srcs:\n  - \"doesNotExist\"",
				},
			},
			Tester: []func(t *testing.T, loc string, state map[string]string) *CommandFixtureTest{
				func(t *testing.T, loc string, state map[string]string) *CommandFixtureTest {
					return &CommandFixtureTest{
						T:    t,
						Args: []string{"describe", "-w", loc, "-o", "json", "pkg1:foo"},
						Eval: func(t *testing.T, stdout, stderr string) {
							var dest pkginfo
							err := json.Unmarshal([]byte(stdout), &dest)
							if err != nil {
								fmt.Println(stdout)
								t.Fatal(err)
							}
							state["v"] = dest.Metadata.Version
						},
					}
				},
				func(t *testing.T, loc string, state map[string]string) *CommandFixtureTest {
					return &CommandFixtureTest{
						T:    t,
						Args: []string{"describe", "-w", loc, "-o", "json", "pkg1:foo"},
						Eval: func(t *testing.T, stdout, stderr string) {
							var dest pkginfo
							err := json.Unmarshal([]byte(stdout), &dest)
							if err != nil {
								fmt.Println(stdout)
								t.Fatal(err)
							}
							if state["v"] != dest.Metadata.Version {
								t.Errorf("component change did change package version")
							}
						},
					}
				},
			},
		},
		{
			Name: "dependency def change changes version",
			Layouts: []map[string]string{
				{
					"WORKSPACE.yaml":  "",
					"pkg1/BUILD.yaml": "packages:\n- name: foo\n  type: generic\n  srcs:\n  - \"doesNotExist\"\n- name: bar\n  type: generic\n  srcs:\n  - \"doesNotExist\"\n  deps:\n  - :foo",
				},
				{
					"pkg1/BUILD.yaml": "packages:\n- name: foo\n  type: generic\n  srcs:\n  - \"alsoDoesNotExist\"\n- name: bar\n  type: generic\n  srcs:\n  - \"doesNotExist\"\n  deps:\n  - :foo",
				},
			},
			Tester: []func(t *testing.T, loc string, state map[string]string) *CommandFixtureTest{
				func(t *testing.T, loc string, state map[string]string) *CommandFixtureTest {
					return &CommandFixtureTest{
						T:    t,
						Args: []string{"describe", "-w", loc, "-o", "json", "pkg1:foo"},
						Eval: func(t *testing.T, stdout, stderr string) {
							var dest pkginfo
							err := json.Unmarshal([]byte(stdout), &dest)
							if err != nil {
								fmt.Println(stdout)
								t.Fatal(err)
							}
							state["v"] = dest.Metadata.Version
						},
					}
				},
				func(t *testing.T, loc string, state map[string]string) *CommandFixtureTest {
					return &CommandFixtureTest{
						T:    t,
						Args: []string{"describe", "-w", loc, "-o", "json", "pkg1:bar"},
						Eval: func(t *testing.T, stdout, stderr string) {
							var dest pkginfo
							err := json.Unmarshal([]byte(stdout), &dest)
							if err != nil {
								fmt.Println(stdout)
								t.Fatal(err)
							}
							if state["v"] == dest.Metadata.Version {
								t.Errorf("dependency def change didn't change version")
							}
						},
					}
				},
			},
		},
		{
			Name: "build args dont change version",
			Layouts: []map[string]string{
				{
					"WORKSPACE.yaml":  "",
					"pkg1/BUILD.yaml": "packages:\n- name: foo\n  type: generic\n  srcs:\n  - \"doesNotExist\"\n  config:\n    commands:\n    - [\"echo\", \"${msg}\"]",
				},
				{},
			},
			Tester: []func(t *testing.T, loc string, state map[string]string) *CommandFixtureTest{
				func(t *testing.T, loc string, state map[string]string) *CommandFixtureTest {
					return &CommandFixtureTest{
						T:    t,
						Args: []string{"describe", "-Dmsg=foo", "-w", loc, "-o", "json", "pkg1:foo"},
						Eval: func(t *testing.T, stdout, stderr string) {
							var dest pkginfo
							err := json.Unmarshal([]byte(stdout), &dest)
							if err != nil {
								fmt.Println(stdout)
								t.Fatal(err)
							}
							state["v"] = dest.Metadata.Version
						},
					}
				},
				func(t *testing.T, loc string, state map[string]string) *CommandFixtureTest {
					return &CommandFixtureTest{
						T:    t,
						Args: []string{"describe", "-Dmsg=bar", "-w", loc, "-o", "json", "pkg1:foo"},
						Eval: func(t *testing.T, stdout, stderr string) {
							var dest pkginfo
							err := json.Unmarshal([]byte(stdout), &dest)
							if err != nil {
								fmt.Println(stdout)
								t.Fatal(err)
							}
							if state["v"] != dest.Metadata.Version {
								t.Errorf("build arg did change version")
							}
						},
					}
				},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.Name, func(t *testing.T) {
			loc, err := os.MkdirTemp("", "pkgdeftest-*")
			if err != nil {
				t.Fatalf("cannot create temporary dir: %q", err)
			}

			state := make(map[string]string)
			for i, l := range test.Layouts {
				for k, v := range l {
					err := os.MkdirAll(filepath.Join(loc, filepath.Dir(k)), 0755)
					if err != nil && !os.IsExist(err) {
						t.Fatalf("cannot create filesystem layout: %q", err)
					}
					err = os.WriteFile(filepath.Join(loc, k), []byte(v), 0644)
					if err != nil && !os.IsExist(err) {
						t.Fatalf("cannot create filesystem layout: %q", err)
					}
				}

				tester := test.Tester[i](t, loc, state)
				tester.Name = fmt.Sprintf("test-%003d", i)
				tester.Run()
			}
		})
	}
}
