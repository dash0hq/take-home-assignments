package main

import "testing"

// TestCanonicalMetadataKey_Deterministic verifies identical inputs always
// produce the same key and the same MetadataID.
func TestCanonicalMetadataKey_Deterministic(t *testing.T) {
	resAttrs := map[string]string{"service.name": "checkout", "host.name": "host-a"}
	scopeAttrs := map[string]string{"lib": "otel-go"}
	dpAttrs := map[string]string{"cpu": "0", "state": "user"}

	key1 := canonicalMetadataKey("cpu.util", "gauge", "1", "scope", "1.0", resAttrs, scopeAttrs, dpAttrs)
	key2 := canonicalMetadataKey("cpu.util", "gauge", "1", "scope", "1.0", resAttrs, scopeAttrs, dpAttrs)

	if key1 != key2 {
		t.Fatalf("expected identical inputs to produce identical canonical keys, got %q vs %q", key1, key2)
	}
	if computeMetadataID(key1) != computeMetadataID(key2) {
		t.Fatalf("expected identical canonical keys to hash to the same MetadataID")
	}
}

// TestCanonicalMetadataKey_MapOrderIndependent verifies map content, not
// insertion order, determines the key - Go map iteration order is randomized.
func TestCanonicalMetadataKey_MapOrderIndependent(t *testing.T) {
	a := map[string]string{"region": "eu", "az": "eu-west-1a", "cpu": "0"}
	b := map[string]string{"cpu": "0", "region": "eu", "az": "eu-west-1a"}

	key1 := canonicalMetadataKey("m", "gauge", "1", "s", "1.0", map[string]string{}, map[string]string{}, a)
	key2 := canonicalMetadataKey("m", "gauge", "1", "s", "1.0", map[string]string{}, map[string]string{}, b)

	if key1 != key2 {
		t.Fatalf("expected map content (not insertion order) to determine the canonical key, got %q vs %q", key1, key2)
	}
}

// TestCanonicalMetadataKey_DifferentInputsDiffer verifies changing any single
// input (name, type, unit, scope, or any attribute map) produces a different, unique key.
func TestCanonicalMetadataKey_DifferentInputsDiffer(t *testing.T) {
	base := map[string]string{"cpu": "0"}
	empty := map[string]string{}

	cases := []struct {
		name string
		key  string
	}{
		{"base", canonicalMetadataKey("cpu.util", "gauge", "1", "scope", "1.0", empty, empty, base)},
		{"different metric name", canonicalMetadataKey("mem.util", "gauge", "1", "scope", "1.0", empty, empty, base)},
		{"different metric type", canonicalMetadataKey("cpu.util", "sum", "1", "scope", "1.0", empty, empty, base)},
		{"different unit", canonicalMetadataKey("cpu.util", "gauge", "%", "scope", "1.0", empty, empty, base)},
		{"different scope name", canonicalMetadataKey("cpu.util", "gauge", "1", "other-scope", "1.0", empty, empty, base)},
		{"different scope version", canonicalMetadataKey("cpu.util", "gauge", "1", "scope", "2.0", empty, empty, base)},
		{"different datapoint attrs", canonicalMetadataKey("cpu.util", "gauge", "1", "scope", "1.0", empty, empty, map[string]string{"cpu": "1"})},
		{"different resource attrs", canonicalMetadataKey("cpu.util", "gauge", "1", "scope", "1.0", map[string]string{"host": "a"}, empty, base)},
		{"different scope attrs", canonicalMetadataKey("cpu.util", "gauge", "1", "scope", "1.0", empty, map[string]string{"lib": "x"}, base)},
	}

	baseKey := cases[0].key
	seen := map[string]string{"base": baseKey}
	for _, c := range cases[1:] {
		if c.key == baseKey {
			t.Errorf("expected %q to produce a different canonical key than base, both were %q", c.name, c.key)
		}
		if existing, ok := seen[c.key]; ok {
			t.Errorf("expected %q to produce a unique canonical key, collided with %q", c.name, existing)
		}
		seen[c.key] = c.name
	}
}

// TestCanonicalMetadataKey_NulSeparatorAvoidsAmbiguity verifies the NUL
// separator prevents key/value concatenation collisions, e.g. {a:bc} vs {ab:c}.
func TestCanonicalMetadataKey_NulSeparatorAvoidsAmbiguity(t *testing.T) {
	attrsA := map[string]string{"a": "bc"}
	attrsB := map[string]string{"ab": "c"}

	keyA := canonicalMetadataKey("m", "gauge", "1", "s", "1.0", map[string]string{}, map[string]string{}, attrsA)
	keyB := canonicalMetadataKey("m", "gauge", "1", "s", "1.0", map[string]string{}, map[string]string{}, attrsB)

	if keyA == keyB {
		t.Fatalf("expected NUL-separated encoding to distinguish {a:bc} from {ab:c}, both produced %q", keyA)
	}
}

// TestComputeMetadataID_NotTriviallyZeroOrConstant verifies the hash isn't
// degenerate - never zero, and different inputs produce different IDs.
func TestComputeMetadataID_NotTriviallyZeroOrConstant(t *testing.T) {
	id1 := computeMetadataID("some-canonical-key")
	id2 := computeMetadataID("a-different-canonical-key")

	if id1 == 0 || id2 == 0 {
		t.Errorf("expected non-zero MetadataIDs, got %d and %d", id1, id2)
	}
	if id1 == id2 {
		t.Errorf("expected different canonical keys to hash to different IDs, both got %d", id1)
	}
}
