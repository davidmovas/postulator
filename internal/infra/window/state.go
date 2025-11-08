package window

import (
	"sync/atomic"
)

var windowIsOpen int32

func SetWindowOpen(v bool) {
	if v {
		atomic.StoreInt32(&windowIsOpen, 1)
	} else {
		atomic.StoreInt32(&windowIsOpen, 0)
	}
}

func IsWindowOpen() bool {
	return atomic.LoadInt32(&windowIsOpen) == 1
}
