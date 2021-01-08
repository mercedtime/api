package app

import "testing"

func BenchmarkStringConcat(b *testing.B) {
	for i := 0; i < b.N; i++ {

	}
}

func BenchmarkStringBuilder(b *testing.B) {
	for i := 0; i < b.N; i++ {

	}
}
