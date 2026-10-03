package driver

import (
	"bytes"
	"encoding/json"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/eugenioenko/goalchemy/internal/diagnostics"
	"github.com/eugenioenko/goalchemy/internal/link"
)

type formatter struct {
	tool string
	args []string
}

type targetFormat struct {
	exts       []string
	formatters []formatter
}

var formats = map[string]targetFormat{
	"go":         {[]string{".go"}, []formatter{{"gofmt", []string{"-w"}}}},
	"typescript": {[]string{".ts"}, []formatter{{"prettier", []string{"--write", "--log-level=warn"}}}},
	"python":     {[]string{".py"}, []formatter{{"ruff", []string{"format", "--quiet"}}, {"black", []string{"--quiet"}}}},
	"java":       {[]string{".java"}, []formatter{{"google-java-format", []string{"--replace"}}}},
	"csharp":     {[]string{".cs"}, []formatter{{"csharpier", []string{"format"}}}},
	"rust":       {[]string{".rs"}, []formatter{{"rustfmt", []string{"--edition", "2021"}}}},
	"c":          {[]string{".c", ".h"}, []formatter{{"clang-format", []string{"-i"}}}},
}

var sourceMapComment = regexp.MustCompile(`(?m)^//# sourceMappingURL=.*\n?`)

// Format rewrites the source files in a target output directory with the
// target's conventional formatter. Generated line tables and source maps no
// longer describe the rewritten files, so they are removed.
func Format(target, out string) []diagnostics.Diagnostic {
	tf, ok := formats[target]
	if !ok {
		return nil
	}
	env := ToolEnv()
	f, path := findFormatter(tf.formatters, env)
	if path == "" {
		var names []string
		for _, f := range tf.formatters {
			names = append(names, f.tool)
		}
		return []diagnostics.Diagnostic{{Code: "GCE008", Severity: diagnostics.Warning, Feature: "formatting",
			Message: "no " + target + " formatter found; output left unformatted",
			Remedy:  "Install " + strings.Join(names, " or ") + " on PATH, or omit -fmt."}}
	}
	files, err := sourceFiles(out, tf.exts)
	if err != nil {
		return formatErr(err.Error())
	}
	if len(files) == 0 {
		return nil
	}
	cmd := exec.Command(path, append(append([]string{}, f.args...), files...)...)
	cmd.Dir, cmd.Env = out, env
	var stderr bytes.Buffer
	cmd.Stdout, cmd.Stderr = &stderr, &stderr
	if err := cmd.Run(); err != nil {
		msg := strings.TrimSpace(stderr.String())
		if msg == "" {
			msg = err.Error()
		}
		return formatErr(f.tool + " failed: " + msg)
	}
	if err := dropPositionMaps(out); err != nil {
		return formatErr(err.Error())
	}
	return nil
}

func formatErr(msg string) []diagnostics.Diagnostic {
	return []diagnostics.Diagnostic{{Code: "GCE008", Severity: diagnostics.Error, Feature: "formatting",
		Message: msg, Remedy: "Rerun without -fmt; the unformatted output is valid."}}
}

func findFormatter(cands []formatter, env []string) (formatter, string) {
	var pathList string
	for _, kv := range env {
		if v, ok := strings.CutPrefix(kv, "PATH="); ok {
			pathList = v
		}
	}
	if tc := ToolchainRoot(); tc != "" {
		if llvm, _ := filepath.Glob(filepath.Join(tc, "llvm-*", "bin")); len(llvm) > 0 {
			sort.Strings(llvm)
			pathList += string(os.PathListSeparator) + llvm[len(llvm)-1]
		}
	}
	for _, f := range cands {
		for _, dir := range filepath.SplitList(pathList) {
			p := filepath.Join(dir, f.tool)
			if st, err := os.Stat(p); err == nil && !st.IsDir() && st.Mode()&0o111 != 0 {
				return f, p
			}
		}
	}
	return formatter{}, ""
}

func sourceFiles(out string, exts []string) ([]string, error) {
	var files []string
	err := filepath.WalkDir(out, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			switch d.Name() {
			case "node_modules", "target", "bin", "obj", "dist", "__pycache__":
				return filepath.SkipDir
			}
			return nil
		}
		for _, e := range exts {
			if strings.HasSuffix(p, e) {
				rel, _ := filepath.Rel(out, p)
				files = append(files, rel)
				break
			}
		}
		return nil
	})
	sort.Strings(files)
	return files, err
}

func dropPositionMaps(out string) error {
	mpath := filepath.Join(out, "goalchemy.manifest.json")
	raw, err := os.ReadFile(mpath)
	if err != nil {
		return err
	}
	var m link.Manifest
	if err := json.Unmarshal(raw, &m); err != nil {
		return err
	}
	kept := m.GeneratedFiles[:0]
	for _, name := range m.GeneratedFiles {
		p := filepath.Join(out, name)
		switch {
		case strings.HasSuffix(name, ".lines") || strings.HasSuffix(name, ".ts.map"):
			if err := os.Remove(p); err != nil && !os.IsNotExist(err) {
				return err
			}
			continue
		case strings.HasSuffix(name, ".ts"):
			src, err := os.ReadFile(p)
			if err != nil {
				return err
			}
			if stripped := sourceMapComment.ReplaceAll(src, nil); !bytes.Equal(stripped, src) {
				if err := os.WriteFile(p, stripped, 0o644); err != nil {
					return err
				}
			}
		}
		kept = append(kept, name)
	}
	m.GeneratedFiles = kept
	data, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return err
	}
	return link.WriteFile(out, "goalchemy.manifest.json", append(data, '\n'))
}
