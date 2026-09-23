package tcgmatch

import (
	"slices"
	"testing"
)

// TestSetAliasesDropTheCode Covers the Gap the Probe Showed: TCGMatch Writes
// `ME05: Pitch Black` and a Store Writes `Mega Evolution - Pitch Black`.
func TestSetAliasesDropTheCode(t *testing.T) {
	if got := setAliases("ME05: Pitch Black"); !slices.Equal(got, []string{"ME05: Pitch Black", "Pitch Black"}) {
		t.Fatalf("setAliases() = %q", got)
	}
	if got := setAliases("Chaos Origins"); !slices.Equal(got, []string{"Chaos Origins"}) {
		t.Fatalf("setAliases() = %q", got)
	}
	// A Tail too short to Name a Set on its own Stays out of the List.
	if got := setAliases("Sword & Shield: VMA"); !slices.Equal(got, []string{"Sword & Shield: VMA"}) {
		t.Fatalf("setAliases() = %q", got)
	}
}
