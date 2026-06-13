package metadata

import (
	"context"
	"testing"
)

func TestAniListResolve(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	resolver := NewResolver()
	ctx := context.Background()

	// Test a known popular series
	md, err := resolver.Resolve(ctx, "Solo Leveling", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if md == nil {
		t.Fatalf("expected metadata for Solo Leveling, got nil")
	}
	if md.AniListID != 105398 { // Known AniList ID for Solo Leveling
		t.Errorf("expected AniListID 105398, got %d", md.AniListID)
	}
	if md.CanonicalTitle == "" {
		t.Errorf("expected a canonical title, got empty string")
	}

	// Test an unknown series
	md, err = resolver.Resolve(ctx, "ThisIsDefNotARealManga123456789", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if md != nil {
		t.Errorf("expected nil metadata for unknown series, got %+v", md)
	}
}
