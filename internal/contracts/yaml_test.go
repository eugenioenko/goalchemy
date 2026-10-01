package contracts

import "testing"

func TestParseYAMLRejects(t *testing.T) {
	bad := map[string]string{
		"duplicate": "a: 1\na: 2\n",
		"alias":     "a: &x {b: c}\nd: *x\n",
		"merge":     "a: {b: c}\nd:\n  <<: {e: f}\n",
		"tag":       "a: !custom x\n",
		"float":     "a: 1.5\n",
		"nonfinite": "a: .inf\n",
		"bigint":    "a: 99999999999\n",
		"negative":  "a: -1\n",
		"intkey":    "1: a\n",
		"multidoc":  "a: b\n---\nc: d\n",
		"yesbool":   "a: !!bool yes\n",
	}
	for name, src := range bad {
		if _, err := ParseYAML([]byte(src)); err == nil {
			t.Errorf("%s: expected rejection", name)
		}
	}
}

func TestParseYAMLAccepts(t *testing.T) {
	v, err := ParseYAML([]byte("a: [x, \"1\", true, null, 3]\nb: {c: d}\n"))
	if err != nil {
		t.Fatal(err)
	}
	m := v.(map[string]any)
	if m["a"].([]any)[4].(int64) != 3 || m["b"].(map[string]any)["c"] != "d" {
		t.Fatalf("unexpected %v", v)
	}
}
