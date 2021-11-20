package testers

import (
	"path"
	"runtime"
	"testing"
)

func DumpCaller(t *testing.T) {
	_, fn, line, _ := runtime.Caller(2)
	t.Errorf("[ %s:%d ]", path.Base(fn), line)
}
