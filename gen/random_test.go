package gen

import "testing"

func TestGeneratorsProduceValues(t *testing.T) {
	if got := String(12); len(got) != 12 {
		t.Fatalf("String length = %d, want 12", len(got))
	}
	if got := Username(); got == "" {
		t.Fatalf("Username is empty")
	}
	if got := UserAgent(); got == "" {
		t.Fatalf("UserAgent is empty")
	}
	if got := RandInt(5, 5); got != 5 {
		t.Fatalf("RandInt(5, 5) = %d", got)
	}
	if got := Sample([]string{"a"}); got != "a" {
		t.Fatalf("Sample = %q", got)
	}
}
