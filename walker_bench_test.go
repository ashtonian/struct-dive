package gowalker

import (
	"reflect"
	"testing"
)

type BenchStruct struct {
	A int
	B string
	C struct {
		D float64
		E bool
	}
	F []int
	G map[string]string
	H *int
}

var userFunc = func(v reflect.Value, meta ObjMeta) error {
	return nil
}

func BenchmarkWalk(b *testing.B) {
	benchStruct := &BenchStruct{}

	b.Run("default", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_, _ = Walk(benchStruct, userFunc)
		}
	})
}
