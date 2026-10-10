package rt

import nativeos "github.com/eugenioenko/goalchemy/lib/os"

func LibOsWriteFile(name string, data []byte, perm uint32) int {
	return nativeos.WriteFile(name, data, perm)
}
