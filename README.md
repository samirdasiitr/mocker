## Mocker
A library that provides functionality for mocking and patching any arbitrary functions or methods in Go.
It uses bouke/moneky to patch the actual function and provides gomock like features.

### Description
#### NewPatch
The NewPatch function creates a new Mock instance and returns a pointer to it. It is a convenient way to initialize a Mock and immediately call the Patch method.
#### Patch
The Patch method takes a target function as an argument and patches it with the generated replacement function. It assigns the patched function to the patchedFunc field and uses the Patch function from an external package to perform the actual patching.
#### PatchInstance
The PatchInstance method is similar to Patch but is used for patching methods on specific instances of a type. It takes a target instance and the name of the method to be patched.
#### AnyTimes
The AnyTimes method sets the anyTimes flag to indicate that the mocked function can be called any number of times.

#### Times
The Times method sets the times field to specify the expected number of times the mocked function should be called.

#### Return
The Return method allows specifying the return values for each invocation of the mocked function. It takes variadic arguments and converts them to reflect.Value before appending them to the returnValues slice.

#### DoAndReturn
The DoAndReturn method takes a function as an argument and assigns it to the returnFn field. This function will be called instead of returning predefined values when the mocked function is invoked.

#### Unpatch
The Unpatch method unpatches the original function, restoring it to its initial state.

### Example
```
func add(a int64, b int64) int64 {
	return a + b
}

func sum(a, b int64) int64 {
	return add(a, b)
}

func TestMocker(tt *testing.T) {
	mock := Mock{}
	mock.Patch(add).Times(1).Return(int64(0))
	mock.Patch(add).Times(1).Return(int64(1))
	mock.Patch(add).Times(1).Return(int64(2))
	log.Printf("%v", sum(1, 2))
	log.Printf("%v", sum(1, 2))
	log.Printf("%v", sum(1, 2))
	mock.Unpatch()
}
```