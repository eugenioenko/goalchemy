// Package sourcemap writes Source Map v3 files mapping generated lines back
// to Goalchemy source positions.
package sourcemap

import (
	"encoding/json"
	"path/filepath"
	"sort"
	"strings"
)

// Mapping links a generated position to a source position (all zero-based).
type Mapping struct {
	GenLine, GenCol int
	Source          string
	SrcLine, SrcCol int
}

type Map struct {
	File     string
	mappings []Mapping
}

func (m *Map) Add(mp Mapping) { m.mappings = append(m.mappings, mp) }

const b64 = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789+/"

func vlq(b *strings.Builder, v int) {
	u := v << 1
	if v < 0 {
		u = (-v << 1) | 1
	}
	for {
		d := u & 31
		u >>= 5
		if u > 0 {
			d |= 32
		}
		b.WriteByte(b64[d])
		if u == 0 {
			return
		}
	}
}

// JSON renders the map with sources relative to dir.
func (m *Map) JSON(dir string) ([]byte, error) {
	ms := append([]Mapping(nil), m.mappings...)
	sort.SliceStable(ms, func(i, j int) bool {
		if ms[i].GenLine != ms[j].GenLine {
			return ms[i].GenLine < ms[j].GenLine
		}
		return ms[i].GenCol < ms[j].GenCol
	})
	srcIdx := map[string]int{}
	var sources []string
	for _, mp := range ms {
		if _, ok := srcIdx[mp.Source]; !ok {
			srcIdx[mp.Source] = len(sources)
			rel := mp.Source
			if r, err := filepath.Rel(dir, mp.Source); err == nil {
				rel = filepath.ToSlash(r)
			}
			sources = append(sources, rel)
		}
	}
	var b strings.Builder
	line, prevSrc, prevLine, prevCol := 0, 0, 0, 0
	lastLine := -1
	for _, mp := range ms {
		if mp.GenLine == lastLine {
			continue
		}
		for line < mp.GenLine {
			b.WriteByte(';')
			line++
		}
		lastLine = mp.GenLine
		vlq(&b, mp.GenCol)
		si := srcIdx[mp.Source]
		vlq(&b, si-prevSrc)
		vlq(&b, mp.SrcLine-prevLine)
		vlq(&b, mp.SrcCol-prevCol)
		prevSrc, prevLine, prevCol = si, mp.SrcLine, mp.SrcCol
	}
	return json.MarshalIndent(map[string]any{
		"version":  3,
		"file":     m.File,
		"sources":  sources,
		"names":    []string{},
		"mappings": b.String(),
	}, "", " ")
}
