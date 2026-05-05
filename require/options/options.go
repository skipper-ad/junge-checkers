package options

import (
	"fmt"

	junge "github.com/skipper-ad/junge-checkers"
)

type ExitInfo struct {
	Status  junge.Status
	Public  string
	Private string
}

type Option func(*ExitInfo)

func GetExitInfo(public, private string, opts ...Option) ExitInfo {
	info := ExitInfo{
		Status:  junge.StatusMumble,
		Public:  public,
		Private: private,
	}
	for _, opt := range opts {
		opt(&info)
	}
	if info.Private == "" {
		info.Private = info.Public
	}
	return info
}

func Private(value string) Option {
	return func(info *ExitInfo) {
		info.Private = value
	}
}

func Privatef(format string, args ...any) Option {
	return func(info *ExitInfo) {
		info.Private = fmt.Sprintf(format, args...)
	}
}

func OK() Option {
	return Status(junge.StatusOK)
}

func Corrupt() Option {
	return Status(junge.StatusCorrupt)
}

func Mumble() Option {
	return Status(junge.StatusMumble)
}

func Down() Option {
	return Status(junge.StatusDown)
}

func CheckFailed() Option {
	return Status(junge.StatusCheckFailed)
}

func Status(status junge.Status) Option {
	return func(info *ExitInfo) {
		info.Status = status
	}
}
