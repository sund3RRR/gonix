package util

import "testing"

func TestInterruptState(t *testing.T) {
	InterruptClear()
	t.Cleanup(InterruptClear)

	if InterruptRequested() {
		t.Fatal("InterruptRequested() = true after clear")
	}

	InterruptRequest()
	if !InterruptRequested() {
		t.Fatal("InterruptRequested() = false after request")
	}

	InterruptClear()
	if InterruptRequested() {
		t.Fatal("InterruptRequested() = true after second clear")
	}
}
