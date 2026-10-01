package driver

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// ToolEnv returns the environment for running target toolchains: toolchains
// unpacked under a .toolchains directory in the working directory or one of
// its parents are put first on PATH, and JAVA_HOME and DOTNET_ROOT are set
// when unset.
func ToolEnv() []string {
	env := os.Environ()
	dir, err := os.Getwd()
	if err != nil {
		return env
	}
	for {
		tc := filepath.Join(dir, ".toolchains")
		if st, err := os.Stat(tc); err == nil && st.IsDir() {
			return withToolchains(env, tc)
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return env
		}
		dir = parent
	}
}

func withToolchains(env []string, tc string) []string {
	var bins []string
	set := map[string]string{}
	if jdks, _ := filepath.Glob(filepath.Join(tc, "jdk-*")); len(jdks) > 0 {
		sort.Strings(jdks)
		j := jdks[len(jdks)-1]
		bins = append(bins, filepath.Join(j, "bin"))
		set["JAVA_HOME"] = j
	}
	if st, err := os.Stat(filepath.Join(tc, "dotnet", "dotnet")); err == nil && !st.IsDir() {
		d := filepath.Join(tc, "dotnet")
		bins = append(bins, d)
		set["DOTNET_ROOT"] = d
	}
	if len(bins) == 0 {
		return env
	}
	var out []string
	for _, kv := range env {
		k, v, _ := strings.Cut(kv, "=")
		switch {
		case k == "PATH":
			kv = "PATH=" + strings.Join(bins, string(os.PathListSeparator)) + string(os.PathListSeparator) + v
		case set[k] != "":
			delete(set, k)
		}
		out = append(out, kv)
	}
	for k, v := range set {
		out = append(out, k+"="+v)
	}
	return out
}
