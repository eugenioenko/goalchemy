package rt

import nativeclock "github.com/eugenioenko/goalchemy/lib/clock"

func LibClockUnix() int64 { return nativeclock.Unix() }
