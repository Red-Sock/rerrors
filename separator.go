package rerrors

import (
	"sync/atomic"
)

const DefaultSeparator = ';'

var separator atomic.Value

func init() {
	SetSeparator(DefaultSeparator)
}

func SetSeparator(sep byte) {
	separator.Store(string(sep))
}

func GetSeparator() string {
	return separator.Load().(string)
}
