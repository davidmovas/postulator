package notifyicon

import (
	"sync"
)

var (
	icon []byte
	once sync.Once
)

func SetTempIcons(png []byte) {
	once.Do(func() {
		icon = png
	})
}

func Icon() []byte {
	return icon
}
