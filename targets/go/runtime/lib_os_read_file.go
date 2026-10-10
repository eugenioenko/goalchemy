package rt

import nativeos "github.com/eugenioenko/goalchemy/lib/os"

func LibOsReadFile(name string) ([]byte, int) { return nativeos.ReadFile(name) }
