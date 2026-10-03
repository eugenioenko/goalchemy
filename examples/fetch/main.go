// goalchemy:network

// Command fetch downloads the iris dataset over HTTPS and summarizes it.
package main

import (
	"github.com/eugenioenko/goalchemy/lib/context"
	"github.com/eugenioenko/goalchemy/lib/errors"
	"github.com/eugenioenko/goalchemy/lib/http"
	"github.com/eugenioenko/goalchemy/std/sort"
	"github.com/eugenioenko/goalchemy/std/strconv"
	"github.com/eugenioenko/goalchemy/std/strings"
)

const url = "https://raw.githubusercontent.com/mwaskom/seaborn-data/71e2436a092d714350de0fc409ca8a8714e7e78f/iris.csv"

type Flower struct {
	Row         int
	SepalLength int
	PetalLength int
	Species     string
}

type Species struct {
	Name     string
	Count    int
	PetalSum int
	PetalMin int
	PetalMax int
}

type byLargestSepal []Flower

func (s byLargestSepal) Len() int { return len(s) }
func (s byLargestSepal) Less(i, j int) bool {
	if s[i].SepalLength != s[j].SepalLength {
		return s[i].SepalLength > s[j].SepalLength
	}
	return s[i].Row < s[j].Row
}
func (s byLargestSepal) Swap(i, j int) { s[i], s[j] = s[j], s[i] }

func tenths(s string) (int, error) {
	whole, frac, _ := strings.Cut(s, ".")
	if len(frac) > 1 {
		return 0, errors.New("too many decimals: " + s)
	}
	if frac == "" {
		frac = "0"
	}
	w, err := strconv.Atoi(whole)
	if err != nil {
		return 0, err
	}
	f, err := strconv.Atoi(frac)
	if err != nil {
		return 0, err
	}
	return w*10 + f, nil
}

func decimal(hundredths int) string {
	frac := strconv.Itoa(hundredths % 100)
	if len(frac) < 2 {
		frac = "0" + frac
	}
	return strconv.Itoa(hundredths/100) + "." + frac
}

func parse(csv string) ([]Flower, error) {
	lines := strings.Split(strings.TrimSpace(csv), "\n")
	if len(lines) == 0 || lines[0] != "sepal_length,sepal_width,petal_length,petal_width,species" {
		return nil, errors.New("unexpected header")
	}
	var out []Flower
	for i, line := range lines[1:] {
		cols := strings.Split(line, ",")
		if len(cols) != 5 {
			return nil, errors.New("row " + strconv.Itoa(i+1) + ": want 5 columns")
		}
		sepal, err := tenths(cols[0])
		if err != nil {
			return nil, err
		}
		petal, err := tenths(cols[2])
		if err != nil {
			return nil, err
		}
		out = append(out, Flower{Row: i + 1, SepalLength: sepal, PetalLength: petal, Species: cols[4]})
	}
	return out, nil
}

func main() {
	status, _, body, err := http.Do(context.Background(), "GET", url, []string{"Accept", "text/csv"}, nil, 1<<20, 30000)
	if err != nil {
		println("fetch failed:", err.Error())
		return
	}
	if status != 200 {
		println("fetch failed: status", status)
		return
	}
	println("fetched", len(body), "bytes")

	flowers, err := parse(string(body))
	if err != nil {
		println("parse failed:", err.Error())
		return
	}
	println("rows:", len(flowers))

	var groups []*Species
	index := map[string]*Species{}
	for _, f := range flowers {
		g := index[f.Species]
		if g == nil {
			g = &Species{Name: f.Species, PetalMin: f.PetalLength, PetalMax: f.PetalLength}
			index[f.Species] = g
			groups = append(groups, g)
		}
		g.Count++
		g.PetalSum += f.PetalLength
		if f.PetalLength < g.PetalMin {
			g.PetalMin = f.PetalLength
		}
		if f.PetalLength > g.PetalMax {
			g.PetalMax = f.PetalLength
		}
	}
	println("petal length (cm) by species:")
	for _, g := range groups {
		println(" ", g.Name, "n="+strconv.Itoa(g.Count), "min="+decimal(g.PetalMin*10), "max="+decimal(g.PetalMax*10), "mean="+decimal(g.PetalSum*10/g.Count))
	}

	sorted := append([]Flower(nil), flowers...)
	sort.Sort(byLargestSepal(sorted))
	println("longest sepals:")
	for _, f := range sorted[:5] {
		println("  row", f.Row, f.Species, decimal(f.SepalLength*10), "cm")
	}
}
