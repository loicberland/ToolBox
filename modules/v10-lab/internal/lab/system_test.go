package lab

import (
	"strings"
	"testing"
)

func TestCheckCreateTargetAvailableExistingWithoutOverwrite(t *testing.T) {
	target := t.TempDir()

	err := checkCreateTargetAvailable(target, false)
	if err == nil {
		t.Fatal("expected existing target error")
	}
	if !strings.Contains(err.Error(), "overwrite=false") || !strings.Contains(err.Error(), target) {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestCheckCreateTargetAvailableExistingWithOverwrite(t *testing.T) {
	target := t.TempDir()

	if err := checkCreateTargetAvailable(target, true); err != nil {
		t.Fatalf("expected existing target with overwrite to be accepted, got %v", err)
	}
}

func TestCheckCreateTargetAvailableMissingWithoutOverwrite(t *testing.T) {
	target := t.TempDir() + "-missing"

	if err := checkCreateTargetAvailable(target, false); err != nil {
		t.Fatalf("expected missing target to be accepted, got %v", err)
	}
}

func TestCheckCreateTargetAvailableEmptyTarget(t *testing.T) {
	err := checkCreateTargetAvailable(" ", false)
	if err == nil {
		t.Fatal("expected empty target error")
	}
	if !strings.Contains(err.Error(), "targetPath vide") {
		t.Fatalf("unexpected error: %v", err)
	}
}
