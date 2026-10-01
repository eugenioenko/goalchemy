package main

type Config struct {
	Name  string
	Limit int
	Tags  []string
}

var counter int
var cfg = Config{Name: "default", Limit: 3}
var registry = map[string]func() int{}
var table [3]int

func register(name string, f func() int) { registry[name] = f }

func init() {
	register("double", func() int { return counter * 2 })
	register("limit", func() int { return cfg.Limit })
}

func bump(p *int) { *p += 1 }

func main() {
	bump(&counter)
	bump(&counter)
	pl := &cfg.Limit
	*pl = 10
	pc := &cfg
	pc.Name = "changed"
	cfg = Config{Name: "reset", Limit: 5}
	*pl += 1
	println(counter, cfg.Name, cfg.Limit, pc.Name, *pl)
	println(registry["double"](), registry["limit"]())
	tp := &table
	tp[1] = 9
	table[2] = 4
	println(table[0], table[1], tp[2])
	cfg.Tags = append(cfg.Tags, "a")
	snapshot := cfg
	snapshot.Tags[0] = "shared"
	snapshot.Name = "copy"
	println(cfg.Tags[0], cfg.Name, snapshot.Name)
}
