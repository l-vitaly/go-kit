package transportlayer

import (
	"runtime"
	"strings"
)

func GetCallerName() string {
	pc, _, _, _ := runtime.Caller(2)
	funcName := runtime.FuncForPC(pc).Name()
	parts := strings.Split(funcName, ".")
	return parts[len(parts)-1]
}
