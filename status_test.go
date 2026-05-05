package junge

import "testing"

func TestStatusCodesMatchChecklib(t *testing.T) {
	tests := map[Status]int{
		StatusOK:          101,
		StatusCorrupt:     102,
		StatusMumble:      103,
		StatusDown:        104,
		StatusCheckFailed: 110,
	}
	for status, want := range tests {
		if got := status.Code(); got != want {
			t.Fatalf("%s code = %d, want %d", status, got, want)
		}
	}
}
