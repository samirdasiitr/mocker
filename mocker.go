package mocker

import (
	"fmt"
	"log"
	"reflect"
	"runtime"
	"strings"
	"sync"
)

type Return struct {
	returnFn     reflect.Value
	returnValues []reflect.Value
	times        int
	calledTimes  int
}

type Mock struct {
	originalFunc reflect.Value
	patchedFunc  reflect.Value
	anyTimes     bool
	returnValues []Return
	isRecording  bool
	recordedArgs [][]interface{}
	name         string
}

type PatchFn struct {
	value    reflect.Value
	refCount int
}

type Mocker struct {
	mocks    map[string]map[string]*Mock
	patchFns map[string]*PatchFn
	sync.Mutex
}

var mockerState Mocker

func init() {
	mockerState.mocks = make(map[string]map[string]*Mock)
	mockerState.patchFns = make(map[string]*PatchFn)
}

// registerMock a mock
func registerMock(testFnName, fnName string, mock *Mock) {
	if _, ok := mockerState.mocks[testFnName][fnName]; ok {
		log.Fatalf("Mock for %q (from %q) is already registered, use previous mock instance",
			fnName, testFnName)
	}
	mockerState.mocks[testFnName][fnName] = mock
}

// NewMock returns a new Mock
func NewMock() *Mock {
	mock := &Mock{
		recordedArgs: make([][]interface{}, 0),
	}
	mockerState.Lock()
	defer mockerState.Unlock()

	testName := getTestFunctionName()
	if testName == "unknown" {
		log.Fatalf("Failed to determine test function name")
	}

	if _, ok := mockerState.mocks[testName]; !ok {
		mockerState.mocks[testName] = make(map[string]*Mock)
	}

	return mock
}

// getTestFunctionName returns the test function of the context.
func getTestFunctionName() string {
	frame := 1
	for {
		// Use the runtime package to get caller information
		pc, _, _, _ := runtime.Caller(frame)
		caller := runtime.FuncForPC(pc)

		// Retrieve the name of the caller function
		if caller != nil {
			tokens := strings.Split(caller.Name(), ".")
			funcName := tokens[len(tokens)-1]
			if strings.HasPrefix(funcName, "Test") {
				return caller.Name()
			}

			if funcName == "goexit" {
				break
			}
		}
		frame++
	}
	return "unknown"
}

func getCallerOFFunc(fn string) string {
	frame := 0
	for {
		// Use the runtime package to get caller information
		pc, _, _, _ := runtime.Caller(frame)
		caller := runtime.FuncForPC(pc)

		// Retrieve the name of the caller function
		if caller != nil {

			if strings.Contains(caller.Name(), fn) {
				_, file, line, _ := runtime.Caller(frame + 1)
				return fmt.Sprintf("%s:%d", file, line)
			}

			if strings.Contains(caller.Name(), "goexit") {
				break
			}
		}
		frame++
	}
	return "<unknown location>"
}

// checkPatchGenerationRequired if patch generation is required.
// We generate a replacement function for target function only once.
func checkPatchGenerationRequired(name string) bool {
	mockerState.Lock()
	defer mockerState.Unlock()

	if _, ok := mockerState.patchFns[name]; !ok {
		return true
	}

	return false
}

func generateReplacement(m_ *Mock, target reflect.Value) reflect.Value {
	// Store the original function
	m_.originalFunc = target

	// Create a replacement function using reflection
	replacementFunc := reflect.MakeFunc(m_.originalFunc.Type(), func(args []reflect.Value) []reflect.Value {

		testName := getTestFunctionName()
		if testName == "unknown" {
			log.Fatalf("Failed to determine test function name")
		}

		log.Printf("mock %q:%q Called from %q", testName, m_.name, getCallerOFFunc("generateReplacement"))

		mockerState.Lock()
		m := mockerState.mocks[testName][m_.name]
		mockerState.Unlock()

		// Recording incoming values if recording mode is enabled
		if m.isRecording {
			log.Printf("Recording incoming values")
			rArgs := make([]interface{}, len(args))
			for ii, arg := range args {
				rArgs[ii] = arg.Interface()
			}
			m.recordedArgs = append(m.recordedArgs, rArgs)
		}

		// Handle different call scenarios based on the mock settings
		if m.anyTimes {
			// If the function is allowed to be called any number of times

			// Check if it is Return or DoAndReturn
			if reflect.ValueOf(m.returnValues[0].returnFn).IsZero() {
				return m.returnValues[0].returnValues
			} else {
				return m.returnValues[0].returnFn.Call(args)
			}
		} else if m.returnValues[0].calledTimes < m.returnValues[0].times {
			// If the function has a specific call count limit
			m.returnValues[0].calledTimes++

			// Check if it is Return or DoAndReturn
			if reflect.ValueOf(m.returnValues[0].returnFn).IsZero() {
				rets := m.returnValues[0]
				if m.returnValues[0].calledTimes >= m.returnValues[0].times {
					m.returnValues = m.returnValues[1:]
				}
				return rets.returnValues
			} else {
				return m.returnValues[0].returnFn.Call(args)
			}
		}

		// If the function is called unexpectedly
		panic(fmt.Sprintf("Unexpected call to %q ", m.name))
	}).Interface()

	return reflect.ValueOf(replacementFunc)
}

func (m *Mock) Patch(target interface{}) *Mock {
	funcValue := reflect.ValueOf(target)
	m.name = runtime.FuncForPC(funcValue.Pointer()).Name()

	if checkPatchGenerationRequired(m.name) {
		mockerState.Lock()
		defer mockerState.Unlock()

		m.patchedFunc = generateReplacement(m, funcValue)
		Patch(target, m.patchedFunc.Interface())

		mockerState.patchFns[m.name] = &PatchFn{
			value:    m.patchedFunc,
			refCount: 1,
		}
	} else {
		mockerState.Lock()
		defer mockerState.Unlock()

		mockerState.patchFns[m.name].refCount += 1
	}

	registerMock(getTestFunctionName(), m.name, m)

	return m
}

func (m *Mock) PatchInstance(target interface{}, methodName string) *Mock {
	tType := reflect.TypeOf(target)
	tFunc, _ := tType.MethodByName(methodName)
	fnName := runtime.FuncForPC(tFunc.Func.Pointer()).Name()
	m.name = fnName
	if checkPatchGenerationRequired(m.name) {
		mockerState.Lock()
		defer mockerState.Unlock()

		m.patchedFunc = generateReplacement(m, tFunc.Func)
		PatchInstanceMethod(tType, methodName, m.patchedFunc.Interface())

		mockerState.patchFns[m.name] = &PatchFn{
			value:    m.patchedFunc,
			refCount: 1,
		}
	} else {
		mockerState.Lock()
		defer mockerState.Unlock()

		mockerState.patchFns[m.name].refCount += 1
	}

	registerMock(getTestFunctionName(), m.name, m)
	return m
}

func (m *Mock) PatchInternalMethod(fn reflect.Value, target interface{}, methodName string) *Mock {
	tType := reflect.TypeOf(target)
	m.name = fmt.Sprintf("%s.%s", tType.String(), methodName)
	if checkPatchGenerationRequired(m.name) {
		mockerState.Lock()
		defer mockerState.Unlock()

		m.patchedFunc = generateReplacement(m, fn)
		PatchInstanceMethodEx(target, methodName, m.patchedFunc.Interface())

		mockerState.patchFns[m.name] = &PatchFn{
			value:    m.patchedFunc,
			refCount: 1,
		}
	} else {
		mockerState.Lock()
		defer mockerState.Unlock()

		mockerState.patchFns[m.name].refCount += 1
	}

	registerMock(getTestFunctionName(), m.name, m)
	return m
}

func (m *Mock) AnyTimes() *Mock {
	if m.returnValues == nil {
		m.returnValues = make([]Return, 1)
	}
	m.anyTimes = true
	return m
}

func (m *Mock) Times(n int) *Mock {
	if m.returnValues == nil {
		m.returnValues = make([]Return, 1)
		m.returnValues[0].times = n
	} else {
		m.returnValues = append(m.returnValues, Return{
			times: n,
		})
	}
	return m
}

func (m *Mock) Return(rets ...interface{}) *Mock {
	if m.returnValues == nil {
		panic("please set times or anytimes")
	}

	returnValues := make([]reflect.Value, len(rets))
	fnType := m.originalFunc.Type()
	for ii, val := range rets {
		if val == nil {
			returnValues[ii] = reflect.Zero(fnType.Out(ii))
		} else {
			returnValues[ii] = reflect.ValueOf(val)
		}
	}

	m.returnValues[len(m.returnValues)-1].returnValues = returnValues
	return m
}

func (m *Mock) GetRecordedArgs() [][]interface{} {
	return m.recordedArgs
}

func (m *Mock) DoAndReturn(retFn interface{}) *Mock {
	if m.returnValues == nil {
		panic("please set times or anytimes")
	}
	m.returnValues[len(m.returnValues)-1].returnFn = reflect.ValueOf(retFn)
	return m
}

func (m *Mock) Record() *Mock {
	m.isRecording = true
	return m
}

func (m *Mock) Unpatch() {

	for _, retValue := range m.returnValues {
		if retValue.calledTimes != retValue.times {
			log.Fatalf("Patch %q not called enough times (%d != %d)",
				m.name, retValue.calledTimes, retValue.times)
		}
	}

	mockerState.Lock()
	defer mockerState.Unlock()

	if patch, ok := mockerState.patchFns[m.name]; ok {
		if patch.refCount == 1 {
			// If this is the final unpatch for a target fn.
			// then replace the original function.
			Unpatch(m.originalFunc.Interface())

			delete(mockerState.patchFns, m.name)

			testName := getTestFunctionName()
			if testName == "unknown" {
				log.Fatalf("Failed to determine test function name")
			}

			// Unregister the mock.
			delete(mockerState.mocks[testName], m.name)

		} else {
			patch.refCount -= 1
		}
	}
}

// UnpatchAll removes all mocks registered from a test function.
func UnpatchAll() {
	testName := getTestFunctionName()
	if testName == "unknown" {
		log.Fatalf("Failed to determine test function name")
	}

	mockerState.Lock()
	if _, ok := mockerState.mocks[testName]; ok {
		mockerState.Unlock()

		log.Printf("Removing all mocks from %q", testName)
		for _, mock := range mockerState.mocks[testName] {
			mock.Unpatch()
		}
		return

	} else {
		mockerState.Unlock()
	}

	log.Fatalf("No mocks register from %q", testName)
}

// Below feature is not very usefull hence oommented.
// type MockStruct struct {
// 	mocks map[string]*Mock
// }

// func NewMockStruct(target interface{}) *MockStruct {
// 	ms := MockStruct{
// 		mocks: make(map[string]*Mock),
// 	}

// 	structType := reflect.TypeOf(target)
// 	for i := 0; i < structType.NumMethod(); i++ {
// 		method := structType.Method(i)
// 		mock := NewMock()
// 		ms.mocks[method.Name] = mock.PatchInstance(target, method.Name)
// 	}

// 	return &ms
// }

// func (ms *MockStruct) Patch(methodName string) *Mock {
// 	return ms.mocks[methodName]
// }
