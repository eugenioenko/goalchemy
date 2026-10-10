package main

// goalchemy:gate cooperative

import "github.com/eugenioenko/goalchemy/std/encoding/json"

var ready = make(chan string, 1)

type Waits struct{}

func (Waits) MarshalJSON() ([]byte, error) {
	return []byte(`"` + <-ready + `"`), nil
}

type Report struct {
	Name  string `json:"name"`
	Value Waits  `json:"value"`
	Tags  []any  `json:"tags"`
}

func main() {
	done := make(chan []byte)
	go func() {
		b, err := json.Marshal(Report{Name: "r", Tags: []any{1, Waits{}}})
		if err != nil {
			println(err.Error())
		}
		done <- b
	}()
	ready <- "first"
	ready <- "second"
	println(string(<-done))
}
