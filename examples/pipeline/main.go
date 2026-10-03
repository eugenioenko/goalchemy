// Command pipeline fans jobs out to workers under per-job deadlines.
package main

import (
	"github.com/eugenioenko/goalchemy/lib/context"
	"github.com/eugenioenko/goalchemy/lib/sync"
	"github.com/eugenioenko/goalchemy/lib/time"
	"github.com/eugenioenko/goalchemy/std/sort"
	"github.com/eugenioenko/goalchemy/std/strconv"
	"github.com/eugenioenko/goalchemy/std/strings"
)

const (
	workers = 3
	budget  = 200 * time.Millisecond
)

type Job struct {
	ID   int
	Name string
	Cost time.Duration
}

type Result struct {
	ID    int
	Name  string
	Value int
	Err   error
}

type byID []Result

func (r byID) Len() int           { return len(r) }
func (r byID) Less(i, j int) bool { return r[i].ID < r[j].ID }
func (r byID) Swap(i, j int)      { r[i], r[j] = r[j], r[i] }

func checksum(s string) int {
	sum := 0
	for i := 0; i < len(s); i++ {
		sum = (sum*31 + int(s[i])) % 100003
	}
	return sum
}

func process(parent context.Context, j Job) Result {
	ctx, cancel := context.WithTimeout(parent, budget)
	defer cancel()
	done := make(chan int, 1)
	go func() {
		time.Sleep(j.Cost)
		done <- checksum(j.Name)
	}()
	select {
	case v := <-done:
		return Result{ID: j.ID, Name: j.Name, Value: v}
	case <-ctx.Done():
		return Result{ID: j.ID, Name: j.Name, Err: ctx.Err()}
	}
}

func worker(ctx context.Context, jobs <-chan Job, results chan<- Result, wg *sync.WaitGroup) {
	defer wg.Done()
	for j := range jobs {
		results <- process(ctx, j)
	}
}

func main() {
	list := []Job{
		{1, "resize thumbnails", 30 * time.Millisecond},
		{2, "index documents", 10 * time.Millisecond},
		{3, "transcode video", 2 * time.Second},
		{4, "send newsletter", 20 * time.Millisecond},
		{5, "compress logs", 5 * time.Millisecond},
		{6, "rebuild search", 25 * time.Millisecond},
		{7, "warm cache", 15 * time.Millisecond},
		{8, "rotate keys", 10 * time.Millisecond},
	}

	jobs := make(chan Job)
	results := make(chan Result, len(list))
	ctx := context.Background()

	go func() {
		for _, j := range list {
			jobs <- j
		}
		close(jobs)
	}()

	var wg sync.WaitGroup
	wg.Add(workers)
	for i := 0; i < workers; i++ {
		go worker(ctx, jobs, results, &wg)
	}
	wg.Wait()
	close(results)

	var all []Result
	for r := range results {
		all = append(all, r)
	}
	sort.Sort(byID(all))

	ok, failed := 0, 0
	for _, r := range all {
		var line strings.Builder
		line.WriteString("job ")
		line.WriteString(strconv.Itoa(r.ID))
		line.WriteString(" ")
		line.WriteString(r.Name)
		if r.Err != nil {
			failed++
			line.WriteString(": cancelled (")
			line.WriteString(r.Err.Error())
			line.WriteString(")")
		} else {
			ok++
			line.WriteString(": checksum ")
			line.WriteString(strconv.Itoa(r.Value))
		}
		println(line.String())
	}
	println(strconv.Itoa(len(all))+" jobs on", workers, "workers:", ok, "done,", failed, "cancelled")
}
