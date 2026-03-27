package output

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
)

func TestRenderJSON(t *testing.T) {
	var buf bytes.Buffer
	result := Result{
		Data:       map[string]string{"key": "value"},
		Provenance: map[string]Provenance{"key": Observed},
	}
	if err := FRender(&buf, "json", result, nil); err != nil {
		t.Fatal(err)
	}

	var parsed Result
	if err := json.Unmarshal(buf.Bytes(), &parsed); err != nil {
		t.Fatalf("output is not valid JSON: %v", err)
	}
}

func TestRenderTable(t *testing.T) {
	var buf bytes.Buffer
	table := &Table{
		Headers: []string{"NAME", "VALUE"},
		Rows:    [][]string{{"foo", "bar"}, {"baz", "qux"}},
	}
	result := Result{Data: "unused"}
	if err := FRender(&buf, "table", result, table); err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	if !strings.Contains(out, "NAME") {
		t.Errorf("expected header NAME in output: %s", out)
	}
	if !strings.Contains(out, "foo") {
		t.Errorf("expected foo in output: %s", out)
	}
}

func TestRenderUnsupportedFormat(t *testing.T) {
	var buf bytes.Buffer
	err := FRender(&buf, "xml", Result{}, nil)
	if err == nil {
		t.Fatal("expected error for unsupported format")
	}
}
