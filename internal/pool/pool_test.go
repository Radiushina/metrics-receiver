package pool

import "testing"

type resetItem struct {
	n int
}

func (r *resetItem) Reset() {
	if r == nil {
		return
	}
	r.n = 0
}

func TestNew_NilFactoryReturnsZeroValue(t *testing.T) {
	p := New[*resetItem](nil)

	got := p.Get()
	if got != nil {
		t.Fatalf("Get() with nil factory: got %#v, want nil zero-value", got)
	}
}

func TestNew_FactoryCreatesObject(t *testing.T) {
	p := New(func() *resetItem {
		return &resetItem{n: 42}
	})

	got := p.Get()
	if got == nil {
		t.Fatal("Get() with factory: got nil, want non-nil object")
	}
	if got.n != 42 {
		t.Fatalf("Get() with factory: n=%d, want 42", got.n)
	}
}

func TestPutGet_ResetsBeforeReuse(t *testing.T) {
	p := New(func() *resetItem {
		return &resetItem{}
	})

	item := p.Get()
	item.n = 7
	p.Put(item)

	got := p.Get()
	if got == nil {
		t.Fatal("Get() after Put: got nil")
	}
	if got.n != 0 {
		t.Fatalf("Get() after Put: n=%d, want 0 after Reset", got.n)
	}
}
