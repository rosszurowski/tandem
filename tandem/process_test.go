package tandem

import (
	"bytes"
	"io"
	"os"
	"sort"
	"strings"
	"testing"

	"github.com/rosszurowski/tandem/ansi"
	"golang.org/x/exp/slices"
)

func TestGoAPI(t *testing.T) {
	ansi.NoColor = true
	out, err := captureStdout(func() {
		pm, err := New(Config{
			Cmds:   []string{"echo 'hello' && sleep 0.15", "echo 'world' && sleep 0.15"},
			Silent: true,
		})
		if err != nil {
			t.Fatal(err)
		}
		pm.Run()
	})
	if err != nil {
		t.Fatal(err)
	}

	hasCmd1 := strings.Contains(out, "echo.1  hello")
	hasCmd2 := strings.Contains(out, "echo.2  world")
	if !hasCmd1 || !hasCmd2 {
		t.Fatalf("expected output to contain both commands, got %q", out)
	}
}

func TestParseNpmScripts(t *testing.T) {
	pkg := []byte(`
		{
			"scripts": {
				"dev:css": "echo 'css'",
				"dev:js": "echo 'js'",
				"test": "echo 'test'"
			}
		}
	`)
	tests := []struct {
		cmds    []string
		want    []string
		wantErr bool
	}{
		{
			[]string{"npm:*"},
			[]string{"echo 'css'", "echo 'js'", "echo 'test'"},
			false,
		},
		{
			[]string{"npm:dev:*"},
			[]string{"echo 'css'", "echo 'js'"},
			false,
		},
		{
			[]string{"npm:dev:*", "npm:test"},
			[]string{"echo 'css'", "echo 'js'", "echo 'test'"},
			false,
		},
		{
			[]string{"npm:*:js"},
			[]string{"echo 'js'"},
			false,
		},
		{
			[]string{"npm:banana"},
			nil,
			true,
		},
		{
			[]string{"npm:duck:*"},
			nil,
			true,
		},
	}

	for _, tt := range tests {
		cmds, err := parseNpmScripts(pkg, tt.cmds)
		if (err != nil) != tt.wantErr {
			t.Fatalf("parseNpmScripts(%q): got error %v, want error %v", tt.cmds, err, tt.wantErr)
		}
		var got []string
		for _, c := range cmds {
			got = append(got, c.cmd)
		}
		sort.Strings(got)
		sort.Strings(tt.want)
		if !slices.Equal(got, tt.want) {
			t.Fatalf("parseNpmScripts(%q): got %v, want %v", tt.cmds, got, tt.want)
		}
	}
}

func TestWildcardMatch(t *testing.T) {
	tests := []struct {
		pattern, input string
		want           bool
	}{
		{"a", "a", true},
		{"b", "a", false},
		{"*", "a", true},
		{"*", "abcd", true},
		{"hello:*", "hello:world", true},
		{"hello:*", "helloz:world", false},
		{"*:banana", "hello:banana", true},
		{"*:banana", "hello:bananaz", false},
		{"hello:*:jones", "hello:world:jones", true},
		{"hello:*:jones", "hello:world:steve", false},
	}

	for _, tt := range tests {
		got := wildcardMatch(tt.pattern, tt.input)
		if got != tt.want {
			t.Errorf("wildcardMatch(%q, %q) = %v, want %v", tt.pattern, tt.input, got, tt.want)
		}
	}
}

func captureStdout(f func()) (string, error) {
	stdout := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		return "", err
	}
	os.Stdout = w
	f()
	w.Close()
	os.Stdout = stdout
	var buf bytes.Buffer
	io.Copy(&buf, r)
	return buf.String(), nil
}
