package panicinfo

import (
	"runtime"
	"strings"
)

// LocatePanic takes result of recover(), and will return the file name, line
// and function name that triggered the panic
func LocatePanic(r interface{}) (file string, line int, funcName string) {
	defer func() {
		// Be safe in here
		recover()
	}()
	var pc [16]uintptr

	// Need to start 4 deep
	n := runtime.Callers(4, pc[:])
	for _, pc := range pc[:n] {
		fn := runtime.FuncForPC(pc)
		if fn == nil {
			continue
		}
		file, line = fn.FileLine(pc)
		funcName = fn.Name()
		if !strings.HasPrefix(funcName, "runtime.") {
			break
		}
	}

	return file, line, funcName
}
