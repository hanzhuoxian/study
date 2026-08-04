package main

import "testing"

func BenchmarkWorkNoPool(b *testing.B) {
	for b.Loop() {
		workNoPool()
	}
}

func BenchmarkWorkWithPool(b *testing.B) {
	for b.Loop() {
		workWithPool()
	}
}
