package genericpipeline

import (
	"testing"
	"strconv"
)

func TestPipeline(t *testing.T) {

	fn := func(s string) (string, error) {
		return s + s, nil
	}

	fn1 := func(s string) (int, error) {
		i, err := strconv.Atoi(s)
		if err != nil {
			return -1, err
		}

		return i, err
	}

	fn2 := func(n int) (int, error) {
		return n*n, nil
	}

	p, err := Create(fn, fn1, fn2)
	if err != nil {
		t.Error(err)
		return
	}

	p.Input("42")
	vRaw, err := p.Output()
	if err != nil {
		t.Error(err)
		return
	}

	v, ok := vRaw.(int)
	if  !ok {
			t.Error("Unexpected kind of value")
			return
	}

	ExpectedValue := 17994564
	if v != ExpectedValue {
		t.Errorf("Unexpected value (Actual: %d, Expected: %d)", v, ExpectedValue)
		return
	}
}