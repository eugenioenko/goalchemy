package main

import (
	"github.com/eugenioenko/goalchemy/std/errors"
	"github.com/eugenioenko/goalchemy/std/fmt"
	"github.com/eugenioenko/goalchemy/std/os"
)

func report(label string, err error) {
	if err == nil {
		println(label, "ok")
		return
	}
	var pe *os.PathError
	as := errors.As(err, &pe)
	op, path := "", ""
	if as {
		op, path = pe.Op, pe.Path
	}
	println(fmt.Sprintf("%s: %q as=%t op=%q path=%q unwrap=%q notexist=%t exist=%t perm=%t unsupported=%t",
		label, err.Error(), as, op, path, errors.Unwrap(err).Error(),
		errors.Is(err, os.ErrNotExist), errors.Is(err, os.ErrExist), errors.Is(err, os.ErrPermission), errors.Is(err, errors.ErrUnsupported)))
}

func read(name string) {
	data, err := os.ReadFile(name)
	if err != nil {
		println(fmt.Sprintf("read %q nil=%t len=%d", name, data == nil, len(data)))
		report("  error", err)
		return
	}
	println(fmt.Sprintf("read %q nil=%t len=%d %q", name, data == nil, len(data), data))
}

func main() {
	report("write data.txt", os.WriteFile("data.txt", []byte("hello\x00\xff world\n"), 0o644))
	read("data.txt")
	report("overwrite data.txt", os.WriteFile("data.txt", []byte("hi"), 0o600))
	read("data.txt")
	report("write empty", os.WriteFile("empty.bin", nil, 0o644))
	read("empty.bin")

	big := make([]byte, 200000)
	for i := range big {
		big[i] = byte(i * 7)
	}
	report("write big", os.WriteFile("big.bin", big, 0o644))
	back, err := os.ReadFile("big.bin")
	same := err == nil && len(back) == len(big)
	for i := 0; same && i < len(big); i++ {
		same = back[i] == big[i]
	}
	println("big round trip", same)

	read("missing.txt")
	read("")
	read(".")
	read("data.txt/inner")
	read("bad\x00name")
	report("write missing dir", os.WriteFile("missing/x.txt", []byte("x"), 0o644))
	report("write dir", os.WriteFile(".", []byte("x"), 0o644))
	report("write through file", os.WriteFile("data.txt/inner", []byte("x"), 0o644))
	report("write empty name", os.WriteFile("", []byte("x"), 0o644))

	report("write locked", os.WriteFile("locked.txt", []byte("secret"), 0))
	read("locked.txt")
	report("rewrite locked", os.WriteFile("locked.txt", []byte("again"), 0o600))
}
