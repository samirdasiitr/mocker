package mocker

import (
	"log"
	"reflect"
	"testing"
	_ "unsafe"

	"github.com/stretchr/testify/require"
)

func add(a int64, b int64) int64 {
	return a + b
}

func sum(a, b int64) int64 {
	return add(a, b)
}

func sumInstance(a, b int64) int64 {
	test := &Test{}
	return test.unique(a, b)
}

type Test struct{}

func (test *Test) Add(a int64, b int64) int64 {
	return a + b
}

func (test *Test) unique(a int64, b int64) int64 {
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
	t := &Test{}
	mock := NewMock().PatchInternalMethod(reflect.ValueOf(t.unique), t, "unique")
	mock.Times(2).Return(int64(0))
	mock.Times(1).Return(int64(1))
	require.Equal(tt, int64(0), sumInstance(1, 2))
	require.Equal(tt, int64(0), sumInstance(1, 2))
	require.Equal(tt, int64(1), sumInstance(1, 2))
	mock.Unpatch()
}

//go:linkname patchedUnique Test.unique
func patchedUnique(_ *Test, a, b int64) int64 {
	return 0
}

func TestMockerInstanceDoAndReturn(tt *testing.T) {
	t := &Test{}
	mock := NewMock().PatchInternalMethod(reflect.ValueOf(patchedUnique), t, "unique")
	mock.Times(2).DoAndReturn(func(_ *Test, a, b int64) int64 {
		return (a + b) * 100
	})
	require.Equal(tt, int64(300), sumInstance(1, 2))
	mock.Unpatch()
}
