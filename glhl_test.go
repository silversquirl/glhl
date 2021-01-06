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
