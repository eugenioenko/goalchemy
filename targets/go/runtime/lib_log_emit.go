package rt

import nativelog "github.com/eugenioenko/goalchemy/lib/log"

func LibLogEmit(level int, unixNano int64, message string, attrs []string, text string) {
	nativelog.Emit(level, unixNano, message, attrs, text)
}
