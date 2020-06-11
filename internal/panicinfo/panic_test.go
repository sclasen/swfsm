package panicinfo

import (
	"strings"
	"testing"
)

var file, name string
var line int

func TestPanicHandler(t *testing.T) {
	panicFunc()

	// these must be correct
	if !strings.HasSuffix(file, "panic_test.go") || !strings.HasSuffix(name, "panicFunc") {
		t.Errorf("Panic handler did not collect expected required information: file=%s name=%s", file, name)
	}

	// these are best effort
	if line != 31 {
		t.Logf("Warning: Panic handler did not collect expected best-effort information: line=%d", line)
	}
}

func panicFunc() {
	defer func() {
		// Be safe in here
		r := recover()
		file, line, name = LocatePanic(r)
	}()
	panic("lol I paniced")
}
