package mocker // import "bou.ke/monkey"

import (
	"fmt"
	"log"
	"reflect"
	"sync"
	"unsafe"
)

// patch is an applied patch
// needed to undo a patch
type patch struct {
	originalBytes []byte
	replacement   *reflect.Value
}

var (
	lock = sync.Mutex{}

	patches                = make(map[uintptr]patch)
	unexportedFuncsPatches = make(map[uintptr]patch)
)

type value struct {
	_   uintptr
	ptr unsafe.Pointer
}

func getPtr(v reflect.Value) unsafe.Pointer {
	return (*value)(unsafe.Pointer(&v)).ptr
}

type PatchGuard struct {
	target      reflect.Value
	replacement reflect.Value
}

type PatchGuardForUnexported struct {
	target      uintptr
	replacement reflect.Value
}

func (g *PatchGuard) Unpatch() {
	unpatchValue(g.target)
}

func (g *PatchGuard) Restore() {
	patchValue(g.target, g.replacement)
}

// Patch replaces a function with another
func Patch(target, replacement interface{}) *PatchGuard {
	t := reflect.ValueOf(target)
	r := reflect.ValueOf(replacement)
	patchValue(t, r)

	return &PatchGuard{t, r}
}

// PatchInstanceMethod replaces an instance method methodName for the type target with replacement
// Replacement should expect the receiver (of type target) as the first argument
func PatchInstanceMethod(target reflect.Type, methodName string, replacement interface{}) *PatchGuard {
	m, ok := target.MethodByName(methodName)
	if !ok {
		panic(fmt.Sprintf("unknown method %s", methodName))
	}
	r := reflect.ValueOf(replacement)
	patchValue(m.Func, r)

	return &PatchGuard{m.Func, r}
}

// PatchInstanceMethod replaces an instance method methodName for the type target with replacement
// Replacement should expect the receiver (of type target) as the first argument
func PatchInstanceMethodEx(target interface{}, methodName string,
	replacement interface{}) *PatchGuardForUnexported {

	targetName := fmt.Sprintf("(*%s).%s",
		reflect.TypeOf(target).Elem().Name(), methodName)

	fn, err := FindFuncWithName(targetName)
	if err != nil {
		log.Fatalf("Unable to find function %s, err: %v", targetName, err.Error())
	}

	r := reflect.ValueOf(replacement)
	lock.Lock()
	defer lock.Unlock()

	if patch, ok := unexportedFuncsPatches[fn.Entry()]; ok {
		unpatch(fn.Entry(), patch)
	}

	bytes := replaceFunction(fn.Entry(), (uintptr)(getPtr(r)))
	unexportedFuncsPatches[fn.Entry()] = patch{bytes, &r}

	return &PatchGuardForUnexported{fn.Entry(), r}
}

func patchValue(target, replacement reflect.Value) {
	lock.Lock()
	defer lock.Unlock()

	if patch, ok := patches[target.Pointer()]; ok {
		unpatch(target.Pointer(), patch)
	}

	bytes := replaceFunction(target.Pointer(), (uintptr)(getPtr(replacement)))
	patches[target.Pointer()] = patch{bytes, &replacement}
}

// Unpatch removes any monkey patches on target
// returns whether target was patched in the first place
func Unpatch(target interface{}) bool {
	return unpatchValue(reflect.ValueOf(target))
}

// UnpatchInstanceMethod removes the patch on methodName of the target
// returns whether it was patched in the first place
func UnpatchInstanceMethod(target reflect.Type, methodName string) bool {
	m, ok := target.MethodByName(methodName)
	if !ok {
		panic(fmt.Sprintf("unknown method %s", methodName))
	}
	return unpatchValue(m.Func)
}

// UnpatchAll removes all applied monkeypatches
func unpatchAll() {
	lock.Lock()
	defer lock.Unlock()
	for target, p := range patches {
		unpatch(target, p)
		delete(patches, target)
	}
	for target, p := range unexportedFuncsPatches {
		unpatch(target, p)
		delete(unexportedFuncsPatches, target)
	}
}

// Unpatch removes a monkeypatch from the specified function
// returns whether the function was patched in the first place
func unpatchValue(target reflect.Value) bool {
	lock.Lock()
	defer lock.Unlock()
	patch, ok := patches[target.Pointer()]
	if !ok {
		return false
	}
	unpatch(target.Pointer(), patch)
	delete(patches, target.Pointer())
	return true
}

func unpatch(target uintptr, p patch) {
	copyToLocation(target, p.originalBytes)
}
