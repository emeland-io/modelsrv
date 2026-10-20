package eventmgr

import (
	"runtime"
	"strconv"
	"strings"
)

// goroutineID returns the current goroutine's ID.
//
// This is used only to scope a replication stamp to the goroutine running
// ApplyReplicated (see eventManager.eventStamp). The model's EventSink.Receive
// signature carries no context, and threading a stamp parameter through it
// would require changing every sink implementation; goroutine scoping is the
// contained alternative that keeps a concurrent local mutation on another
// goroutine from reading a remote origin. The id is used purely as an ephemeral
// map key within a single synchronous call chain and never persisted or
// compared across time, so goroutine-id reuse is not a correctness concern.
func goroutineID() uint64 {
	var buf [64]byte
	n := runtime.Stack(buf[:], false)
	// The stack header is "goroutine <id> [status]:".
	s := strings.TrimPrefix(string(buf[:n]), "goroutine ")
	if i := strings.IndexByte(s, ' '); i >= 0 {
		s = s[:i]
	}
	id, _ := strconv.ParseUint(s, 10, 64)
	return id
}
