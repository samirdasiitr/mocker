package mocker

import (
	"log"
	"testing"
)

func add(a int64, b int64) int64 {
	return a + b
}

func sum(a, b int64) int64 {
	test := &Test{}
	return test.Add(a, b)
}

func sumInstance(a, b int64) int64 {
	test := &Test{}
	return test.Add(a, b)
}

type Test struct{}

func (test *Test) Add(a int64, b int64) int64 {
	return a + b
}

func TestMocker(tt *testing.T) {
	mock := NewMock().Patch(add)
	mock.Times(2).Return(int64(0))
	mock.Times(1).Return(int64(1))
	// mock.Times(1).Return(int64(19))
	log.Printf("%v", sum(1, 2))
	log.Printf("%v", sum(1, 2))
	log.Printf("%v", sum(1, 2))
	mock.Unpatch()
}

func TestMockerInstance(tt *testing.T) {
	var t *Test
	mock := NewMock().PatchInstance(t, "Add")
	mock.Times(2).Return(int64(0))
	mock.Times(1).Return(int64(1))
	// mock.Times(1).Return(int64(19))
	log.Printf("%v", sumInstance(1, 2))
	log.Printf("%v", sumInstance(1, 2))
	log.Printf("%v", sumInstance(1, 2))
	mock.Unpatch()
}
