package order

var Log = "order-var"
var Count int

func Record(event string) int { Log += event; Count++; return Count }
func init()                   { Log += "/order-init" }
