package rt

import nativelog "github.com/eugenioenko/goalchemy/lib/log"

func LibLogEnabled(level int) bool { return nativelog.Enabled(level) }
