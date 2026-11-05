package main

import "testing"

// The startup log used to print the truncated default for a 4-byte ASN, 59904:666 for AS
// 4200000000, as if it were the community in use.
func TestDefaultCommunityField(t *testing.T) {
	t.Parallel()

	if got := defaultCommunityField(64520); got != "64520:666" {
		t.Errorf("defaultCommunityField(64520) = %q, want %q", got, "64520:666")
	}
	if got := defaultCommunityField(4200000000); got != "none" {
		t.Errorf("defaultCommunityField(4200000000) = %q, want %q", got, "none")
	}
}
