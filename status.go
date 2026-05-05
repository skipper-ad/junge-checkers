package junge

import "fmt"

// Status is a checker verdict and its process exit code.
//
// The numeric values are the Skipper checker contract. They match the
// historical runner format used by existing services and checker workers.
type Status int

const (
	StatusOK          Status = 101
	StatusCorrupt     Status = 102
	StatusMumble      Status = 103
	StatusDown        Status = 104
	StatusCheckFailed Status = 110
)

func (s Status) Code() int {
	return int(s)
}

func (s Status) String() string {
	switch s {
	case StatusOK:
		return "OK"
	case StatusCorrupt:
		return "CORRUPT"
	case StatusMumble:
		return "MUMBLE"
	case StatusDown:
		return "DOWN"
	case StatusCheckFailed:
		return "CHECK FAILED"
	default:
		return fmt.Sprintf("UNKNOWN_%d", int(s))
	}
}

func validStatus(code int) bool {
	switch Status(code) {
	case StatusOK, StatusCorrupt, StatusMumble, StatusDown, StatusCheckFailed:
		return true
	default:
		return false
	}
}
