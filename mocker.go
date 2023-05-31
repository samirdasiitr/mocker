package mocker

import (
	"fmt"
	"reflect"
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
}

func NewMock() *Mock {
	return &Mock{}
}

func (m *Mock) generateReplacement(target reflect.Value) reflect.Value {
	m.originalFunc = target
	replacementFunc := reflect.MakeFunc(m.originalFunc.Type(), func(args []reflect.Value) []reflect.Value {
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
		panic("unexpected")
	}).Interface()

	return reflect.ValueOf(replacementFunc)
}

func (m *Mock) Patch(target interface{}) *Mock {
	m.patchedFunc = m.generateReplacement(reflect.ValueOf(target))
	Patch(target, m.patchedFunc.Interface())
	return m
}

func (m *Mock) PatchInstance(target interface{}, methodName string) *Mock {
	tType := reflect.TypeOf(target)
	tFunc, _ := tType.MethodByName(methodName)
	m.patchedFunc = m.generateReplacement(tFunc.Func)
	PatchInstanceMethod(tType, methodName, m.patchedFunc.Interface())
	return m
}

func (m *Mock) AnyTimes() *Mock {
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
			val = reflect.Zero(fnType.Out(ii)).Interface()
		}
		returnValues[ii] = reflect.ValueOf(val)
	}

	m.returnValues[len(m.returnValues)-1].returnValues = returnValues
	return m
}

func (m *Mock) DoAndReturn(retFn interface{}) *Mock {
	if m.returnValues == nil {
		panic("please set times or anytimes")
	}
	m.returnValues[len(m.returnValues)-1].returnFn = reflect.ValueOf(retFn)
	return m
}

func (m *Mock) Unpatch() {
	Unpatch(m.originalFunc.Interface())
}

func (m *Mock) ExpectCall(args []reflect.Value) {
	// Implement your logic to record the expected call
	fmt.Println("Expected call with args:", args)
}
