package scaleby

import (
	"testing"
)

func TestScaleBy(t *testing.T) {
	r := &Point{1, 2}
	r.ScaleBy(2)
	if r.X != 2 || r.Y != 4 {
		t.Fail()
	}
}
