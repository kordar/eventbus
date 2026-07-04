package eventbus

import (
	"testing"
)

func TestParseTopicPattern_Exact(t *testing.T) {
	p := parseTopicPattern("order.created")
	if p.raw != "order.created" {
		t.Fatalf("expected 'order.created', got %q", p.raw)
	}
	if !p.matches("order.created") {
		t.Fatal("exact pattern should match same topic")
	}
	if p.matches("order.updated") {
		t.Fatal("exact pattern should NOT match different topic")
	}
}

func TestParseTopicPattern_Star(t *testing.T) {
	p := parseTopicPattern("*")
	if !p.matches("anything") {
		t.Fatal("* should match any topic")
	}
	if !p.matches("order.created.v3") {
		t.Fatal("* should match multi-segment topic")
	}
}

func TestParseTopicPattern_PrefixStar(t *testing.T) {
	p := parseTopicPattern("order.*")
	if !p.matches("order.created") {
		t.Fatal("order.* should match order.created")
	}
	if !p.matches("order.updated") {
		t.Fatal("order.* should match order.updated")
	}
	if !p.matches("order.created.v3") {
		t.Fatal("order.* should match order.created.v3 (prefix match)")
	}
	if p.matches("user.created") {
		t.Fatal("order.* should NOT match user.created")
	}
}

func TestParseTopicPattern_SuffixStar(t *testing.T) {
	p := parseTopicPattern("*.created")
	if !p.matches("order.created") {
		t.Fatal("*.created should match order.created")
	}
	if !p.matches("user.created") {
		t.Fatal("*.created should match user.created")
	}
	if !p.matches("a.b.created") {
		t.Fatal("*.created should match a.b.created (suffix match)")
	}
	if p.matches("order.updated") {
		t.Fatal("*.created should NOT match order.updated")
	}
}

func TestParseTopicPattern_SegmentStar(t *testing.T) {
	p := parseTopicPattern("order.*.updated")
	if !p.matches("order.created.updated") {
		t.Fatal("order.*.updated should match order.created.updated")
	}
	if !p.matches("order.deleted.updated") {
		t.Fatal("order.*.updated should match order.deleted.updated")
	}
	if p.matches("order.created.deleted") {
		t.Fatal("order.*.updated should NOT match order.created.deleted")
	}
	if p.matches("order.created") {
		t.Fatal("order.*.updated should NOT match shorter topic")
	}
}

func TestParseTopicPattern_MultiStar(t *testing.T) {
	p := parseTopicPattern("a.*.c.*")
	if !p.matches("a.b.c.d") {
		t.Fatal("a.*.c.* should match a.b.c.d")
	}
	if p.matches("a.b.c") {
		t.Fatal("a.*.c.* should NOT match 3-segment topic")
	}
}

func TestParseTopicPattern_NoWildcardButMismatch(t *testing.T) {
	p := parseTopicPattern("a.b")
	if p.matches("a.b.c") {
		t.Fatal("exact 'a.b' should NOT match 'a.b.c'")
	}
	if p.matches("a") {
		t.Fatal("exact 'a.b' should NOT match 'a'")
	}
}
