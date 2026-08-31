package main

import (
	"sort"
	"strings"

	"github.com/go-faster/city"
)

// canonicalMetadataKey builds a deterministic key for a series' identity -
// attributes are sorted since Go map iteration order is randomized. metricType
// (e.g. "gauge", "sum") is included so a Gauge and Sum sharing an otherwise
// identical name/unit/scope/attributes don't collide on the same MetadataID.
func canonicalMetadataKey(
	metricName, metricType, metricUnit, scopeName, scopeVersion string,
	resourceAttrs, scopeAttrs, dataPointAttrs map[string]string,
) string {
	var b strings.Builder
	b.WriteString(metricName)
	b.WriteByte(0)
	b.WriteString(metricType)
	b.WriteByte(0)
	b.WriteString(metricUnit)
	b.WriteByte(0)
	b.WriteString(scopeName)
	b.WriteByte(0)
	b.WriteString(scopeVersion)
	b.WriteByte(0)
	writeSortedAttrs(&b, resourceAttrs)
	writeSortedAttrs(&b, scopeAttrs)
	writeSortedAttrs(&b, dataPointAttrs)
	return b.String()
}

// writeSortedAttrs writes key=value pairs in sorted order, NUL-separated to avoid ambiguity.
func writeSortedAttrs(b *strings.Builder, attrs map[string]string) {
	keys := make([]string, 0, len(attrs))
	for k := range attrs {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		b.WriteString(k)
		b.WriteByte('=')
		b.WriteString(attrs[k])
		b.WriteByte(0)
	}
}

// computeMetadataID hashes the canonical key into a MetadataID via
// city.Hash64. Doesn't match ClickHouse's own cityHash64() SQL function
// (verified empirically) - Go-only, never recompute this in SQL expecting a match.
func computeMetadataID(canonicalKey string) uint64 {
	return city.Hash64([]byte(canonicalKey))
}
