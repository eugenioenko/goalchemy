package rt

import nativeclock "github.com/eugenioenko/goalchemy/lib/clock"

func LibClockUnixNano() int64 { return nativeclock.UnixNano() }
