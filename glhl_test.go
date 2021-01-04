package glhl

import "testing"

func TestNewContext(t *testing.T) {
	ctx, err := NewContext(3, 3, 0)
	if err != nil {
		t.Fatal(err)
	}
	defer ctx.Destroy()
	ctx.MakeContextCurrent()
}

func TestNewSharedContext(t *testing.T) {
	ctx, err := NewContext(3, 3, 0)
	if err != nil {
		t.Fatal(err)
	}
	defer ctx.Destroy()
	ctx.MakeContextCurrent()

	ctx2, err := NewSharedContext(3, 3, 0, ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer ctx2.Destroy()
	ctx2.MakeContextCurrent()
}
