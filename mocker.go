package mocker

import (
	"fmt"
	"log"
	"reflect"
	"runtime"
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

func NewMock() *Mock {
	return &Mock{
		recordedArgs: make([][]interface{}, 0),
	}
}

func (m *Mock) generateReplacement(target reflect.Value) reflect.Value {
	m.originalFunc = target
	replacementFunc := reflect.MakeFunc(m.originalFunc.Type(), func(args []reflect.Value) []reflect.Value {
		if m.isRecording {
			log.Printf("Recording incoming values")
			rArgs := make([]interface{}, len(args))
			for ii, arg := range args {
				rArgs[ii] = arg.Interface()
			}
			m.recordedArgs = append(m.recordedArgs, rArgs)
		}
		if m.anyTimes {
			if reflect.ValueOf(m.returnValues[0].returnFn).IsZero() {
				return m.returnValues[0].returnValues
			} else {
				return m.returnValues[0].returnFn.Call(args)
			}
		} else if m.returnValues[0].calledTimes < m.returnValues[0].times {
			m.returnValues[0].calledTimes++
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
		panic("unexpected call to " + m.name)
	}).Interface()

	return reflect.ValueOf(replacementFunc)
}

func (m *Mock) Patch(target interface{}) *Mock {
	funcValue := reflect.ValueOf(target)
	m.name = runtime.FuncForPC(funcValue.Pointer()).Name()
	m.patchedFunc = m.generateReplacement(funcValue)
	Patch(target, m.patchedFunc.Interface())
	return m
}

func (m *Mock) PatchInstance(target interface{}, methodName string) *Mock {
	tType := reflect.TypeOf(target)
	m.name = fmt.Sprintf("%s.%s", tType.Name(), methodName)
	tFunc, _ := tType.MethodByName(methodName)
	m.patchedFunc = m.generateReplacement(tFunc.Func)
	PatchInstanceMethod(tType, methodName, m.patchedFunc.Interface())
	return m
}

func (m *Mock) PatchInternalMethod(fn reflect.Value, target interface{}, methodName string) *Mock {
	tType := reflect.TypeOf(target)
	m.name = fmt.Sprintf("%s.%s", tType.Name(), methodName)
	m.patchedFunc = m.generateReplacement(fn)
	PatchInstanceMethodEx(target, methodName, m.patchedFunc.Interface())
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
	Unpatch(m.originalFunc.Interface())
}

type MockStruct struct {
	mocks map[string]*Mock
}

func NewMockStruct(target interface{}) *MockStruct {
	ms := MockStruct{
		mocks: make(map[string]*Mock),
	}

	structType := reflect.TypeOf(target)
	for i := 0; i < structType.NumMethod(); i++ {
		method := structType.Method(i)
		mock := NewMock()
		ms.mocks[method.Name] = mock.PatchInstance(target, method.Name)
	}

	return &ms
}

func (ms *MockStruct) Patch(methodName string) *Mock {
	return ms.mocks[methodName]
}
