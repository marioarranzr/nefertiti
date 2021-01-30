package notify

import (
	"github.com/marioarranzr/nefertiti/flag"
)

const (
	LEVEL_NOTHING = 0
	LEVEL_ERRORS  = 1
	LEVEL_DEFAULT = 2
	LEVEL_VERBOSE = 3
)

type Notification int

const (
	ERROR Notification = iota
	INFO
	FILLED
	OPENED
	CANCELLED
)

func Level() int64 {
	out := int64(LEVEL_DEFAULT)
	flg := flag.Get("notify")
	if flg.Exists {
		out, _ = flg.Int64()
	}
	return out
}

func CanSend(level int64, notification Notification) bool {
	switch level {
	case LEVEL_NOTHING:
		return false
	case LEVEL_ERRORS:
		return notification == ERROR
	case LEVEL_VERBOSE:
		return true
	}
	return notification == ERROR || notification == INFO || notification == FILLED
}
