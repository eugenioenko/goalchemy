package main

import (
	"github.com/eugenioenko/goalchemy/std/encoding/json"
	"github.com/eugenioenko/goalchemy/std/encoding/jsonvalue"
	"github.com/eugenioenko/goalchemy/std/errors"
	"github.com/eugenioenko/goalchemy/std/strconv"
	"github.com/eugenioenko/goalchemy/std/strings"
	"github.com/eugenioenko/goalchemy/std/time"
)

type RuleType int32

const (
	RuleUnspecified RuleType = 0
	RuleAllOf       RuleType = 1
	RuleAnyOf       RuleType = 2
	RuleHierarchy   RuleType = 3
)

var ruleNames = map[RuleType]string{0: "RULE_UNSPECIFIED", 1: "RULE_ALL_OF", 2: "RULE_ANY_OF", 3: "RULE_HIERARCHY"}

func (r RuleType) MarshalJSON() ([]byte, error) {
	if n, ok := ruleNames[r]; ok {
		return json.Marshal(n)
	}
	return json.Marshal(int32(r))
}

func (r *RuleType) UnmarshalJSON(b []byte) error {
	var name string
	if err := json.Unmarshal(b, &name); err == nil {
		for v, n := range ruleNames {
			if n == name {
				*r = v
				return nil
			}
		}
		return errors.New("unknown RuleType " + strconv.Quote(name))
	}
	var n int32
	if err := json.Unmarshal(b, &n); err != nil {
		return err
	}
	*r = RuleType(n)
	return nil
}

type Int64 int64

func (i Int64) MarshalJSON() ([]byte, error) { return json.Marshal(strconv.FormatInt(int64(i), 10)) }

func (i *Int64) UnmarshalJSON(b []byte) error {
	var n json.Number
	if err := json.Unmarshal(b, &n); err != nil {
		return err
	}
	v, err := n.Int64()
	if err != nil {
		return err
	}
	*i = Int64(v)
	return nil
}

type Timestamp struct {
	Seconds int64
	Nanos   int32
}

func (t Timestamp) MarshalJSON() ([]byte, error) {
	return json.Marshal(time.Unix(t.Seconds, int64(t.Nanos)).UTC().Format(time.RFC3339Nano))
}

func (t *Timestamp) UnmarshalJSON(b []byte) error {
	var s string
	if err := json.Unmarshal(b, &s); err != nil {
		return err
	}
	parsed, err := time.Parse(time.RFC3339Nano, s)
	if err != nil {
		return err
	}
	t.Seconds, t.Nanos = parsed.Unix(), int32(parsed.Nanosecond())
	return nil
}

type Duration struct {
	Seconds int64
	Nanos   int32
}

func (d Duration) MarshalJSON() ([]byte, error) {
	s := strconv.FormatInt(d.Seconds, 10)
	if d.Nanos != 0 {
		frac := strconv.Itoa(int(d.Nanos) + 1000000000)[1:]
		s += "." + strings.TrimRight(frac, "0")
	}
	return json.Marshal(s + "s")
}

func (d *Duration) UnmarshalJSON(b []byte) error {
	var s string
	if err := json.Unmarshal(b, &s); err != nil {
		return err
	}
	if !strings.HasSuffix(s, "s") {
		return errors.New("duration must end in s: " + s)
	}
	whole, frac, _ := strings.Cut(strings.TrimSuffix(s, "s"), ".")
	sec, err := strconv.ParseInt(whole, 10, 64)
	if err != nil {
		return err
	}
	d.Seconds = sec
	if frac != "" {
		n, err := strconv.Atoi((frac + "000000000")[:9])
		if err != nil {
			return err
		}
		d.Nanos = int32(n)
	}
	return nil
}

type Any struct {
	TypeURL string
	Value   jsonvalue.Value
}

func (a Any) MarshalJSON() ([]byte, error) {
	fields := []any{"@type", jsonvalue.String(a.TypeURL)}
	for _, k := range a.Value.Keys() {
		fields = append(fields, k, a.Value.Get(k))
	}
	return jsonvalue.Encode(jsonvalue.Object(fields...))
}

func (a *Any) UnmarshalJSON(b []byte) error {
	v, err := jsonvalue.Parse(b)
	if err != nil {
		return err
	}
	a.TypeURL = v.Get("@type").String()
	var fields []any
	for _, k := range v.Keys() {
		if k != "@type" {
			fields = append(fields, k, v.Get(k))
		}
	}
	a.Value = jsonvalue.Object(fields...)
	return nil
}

type Value struct {
	ID    string `json:"id,omitempty"`
	Value string `json:"value,omitempty"`
}

type Attribute struct {
	ID        string            `json:"id,omitempty"`
	Name      string            `json:"name,omitempty"`
	Rule      RuleType          `json:"rule,omitempty"`
	Values    []Value           `json:"values,omitempty"`
	CreatedAt *Timestamp        `json:"createdAt,omitempty"`
	TTL       *Duration         `json:"ttl,omitempty"`
	Version   Int64             `json:"version,omitempty"`
	Labels    map[string]string `json:"labels,omitempty"`
	Metadata  jsonvalue.Value   `json:"metadata"`
	Details   []Any             `json:"details,omitempty"`
	Active    *bool             `json:"active,omitempty"`
}

type ListResponse struct {
	Attributes []Attribute `json:"attributes"`
	Pagination struct {
		NextOffset int32 `json:"nextOffset,omitempty"`
		Total      int32 `json:"total"`
	} `json:"pagination"`
}

func main() {
	input := `{
	  "attributes": [
	    {"id": "a1", "name": "level", "rule": "RULE_HIERARCHY", "values": [{"id": "v1", "value": "top"}],
	     "createdAt": "2024-05-06T07:08:09.123456789Z", "ttl": "90.5s", "version": "9007199254740993",
	     "labels": {"team": "x"}, "metadata": {"z": [1, 2.50, null], "a": {"nested": true}},
	     "details": [{"@type": "type.googleapis.com/policy.Extra", "note": "hi", "n": 3}],
	     "active": false, "unknownField": {"ignored": [1, 2]}},
	    {"id": "a2", "rule": 2, "version": 42, "metadata": null, "createdAt": "2024-01-01T00:00:00+02:00"}
	  ],
	  "pagination": {"total": 2}
	}`
	var resp ListResponse
	err := json.Unmarshal([]byte(input), &resp)
	println("decode", err == nil)
	out, err := json.Marshal(&resp)
	println(string(out), err == nil)
	pretty, _ := json.MarshalIndent(resp.Attributes[1], "", "  ")
	println(string(pretty))
	var back ListResponse
	err = json.Unmarshal(out, &back)
	again, _ := json.Marshal(&back)
	println("round trip", err == nil, string(again) == string(out))

	for _, bad := range []string{
		`{"attributes":[{"rule":"RULE_NOPE"}]}`,
		`{"attributes":[{"createdAt":"yesterday"}]}`,
		`{"attributes":[{"ttl":"5m"}]}`,
		`{"attributes":[{"version":"x"}]}`,
		`{"attributes":[{"version":1.5}]}`,
		`{"attributes":[{"labels":{"a":1}}]}`,
		`{"attributes":[{"id":"ok"}],"pagination":{"total":"2"}}`,
	} {
		var r ListResponse
		err := json.Unmarshal([]byte(bad), &r)
		println("error", err.Error())
	}
}
