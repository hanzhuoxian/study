package example03

import "testing"

var args = []string{"aa", "bb", "cc", "dd", "ee", "ff", "gg", "hh", "ii", "jj", "kk"}

func BenchmarkForJoin(b *testing.B) {
	for i := 0; i < b.N; i++ {
		ForJoin(args)
	}
}

func BenchmarkStringsJoin(b *testing.B) {
	for i := 0; i < b.N; i++ {
		StringsJoin(args)
	}
}
