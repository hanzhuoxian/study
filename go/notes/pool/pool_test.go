package main

import "testing"

func BenchmarkWorkNoPool(b *testing.B) {
	for i := 0; i < b.N; i++ {
		workNoPool()
	}
}

func BenchmarkWorkWithPool(b *testing.B) {
	for i := 0; i < b.N; i++ {
		workWithPool()
	}
}
