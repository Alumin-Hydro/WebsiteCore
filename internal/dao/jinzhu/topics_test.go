package jinzhu

import "testing"

func TestCountPublicTagRefsCountsEachPostOnce(t *testing.T) {
	got := countPublicTagRefs([]string{
		"approved, shared, shared",
		"shared",
		" , approved, ",
		"",
	})

	want := map[string]int64{
		"approved": 2,
		"shared":   2,
	}
	if len(got) != len(want) {
		t.Fatalf("tag count = %d, want %d: %#v", len(got), len(want), got)
	}
	for tag, count := range want {
		if got[tag] != count {
			t.Errorf("count[%q] = %d, want %d", tag, got[tag], count)
		}
	}
}
