package testutil

import "testing"

func TestParseDirectivesGateValue(t *testing.T) {
	for _, tc := range []struct{ src, want string }{
		{"package main", ""},
		{"// goalchemy:gate sequential\npackage main", "sequential"},
		{"// goalchemy:gate cooperative\npackage main", "cooperative"},
	} {
		if got := ParseDirectives(tc.src).Gate; got != tc.want {
			t.Errorf("gate %q, want %q", got, tc.want)
		}
	}
}
