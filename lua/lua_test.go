package lua

import (
	"testing"
	"unsafe"
)

type TestStruct struct {
	IntField    int
	StringField string
	FloatField  float64
}

func TestGoStruct(t *testing.T) {
	L := NewState()
	L.OpenLibs()
	defer L.Close()

	ts := &TestStruct{10, "test", 2.3}

	L.CheckStack(1)

	L.PushGoStruct(ts)
	L.SetGlobal("t")

	L.GetGlobal("t")
	if !L.IsGoStruct(-1) {
		t.Fatal("Not go struct")
	}

	tsr := L.ToGoStruct(-1).(*TestStruct)
	if tsr != ts {
		t.Fatal("Retrieved something different from what we inserted")
	}

	L.Pop(1)

	L.PushString("This is not a struct")
	if L.ToGoStruct(-1) != nil {
		t.Fatal("Non-GoStruct value attempted to convert into GoStruct should result in nil")
	}

	L.Pop(1)
}

func TestCheckStringSuccess(t *testing.T) {
	L := NewState()
	L.OpenLibs()
	defer L.Close()

	Test := func(L *State) int {
		L.PushString("this is a test")
		L.CheckString(-1)
		return 0
	}

	L.Register("test", Test)
	err := L.DoString("test()")
	if err != nil {
		t.Fatalf("DoString did return an error: %v\n", err.Error())
	}
}

func TestCheckStringFail(t *testing.T) {
	L := NewState()
	L.OpenLibs()
	defer L.Close()

	Test := func(L *State) int {
		L.CheckString(-1)
		return 0
	}

	L.Register("test", Test)
	err := L.DoString("test();")
	if err == nil {
		t.Fatal("DoString did not return an error\n")
	}
}

func TestPCallHidden(t *testing.T) {
	L := NewState()
	L.OpenLibs()
	defer L.Close()

	err := L.DoString("pcall(print, \"ciao\")")
	if err == nil {
		t.Fatal("Can use pcall\n")
	}

	err = L.DoString("unsafe_pcall(print, \"ciao\")")
	if err != nil {
		t.Fatal("Can not use unsafe_pcall\n")
	}
}

func TestCall(t *testing.T) {
	L := NewState()
	L.OpenLibs()
	defer L.Close()

	test := func(L *State) int {
		arg1 := L.ToString(1)
		arg2 := L.ToString(2)
		arg3 := L.ToString(3)

		if arg1 != "Argument1" {
			t.Fatal("Got wrong argument (1)")
		}

		if arg2 != "Argument2" {
			t.Fatal("Got wrong argument (2)")
		}

		if arg3 != "Argument3" {
			t.Fatal("Got wrong argument (3)")
		}

		L.PushString("Return1")
		L.PushString("Return2")

		return 2
	}

	L.Register("test", test)

	L.PushString("Dummy")
	L.GetGlobal("test")
	L.PushString("Argument1")
	L.PushString("Argument2")
	L.PushString("Argument3")
	err := L.Call(3, 2)
	if err != nil {
		t.Fatalf("Error executing call: %v\n", err)
	}

	dummy := L.ToString(1)
	ret1 := L.ToString(2)
	ret2 := L.ToString(3)

	if dummy != "Dummy" {
		t.Fatal("The stack was disturbed")
	}

	if ret1 != "Return1" {
		t.Fatalf("Wrong return value (1) got: <%s>", ret1)
	}

	if ret2 != "Return2" {
		t.Fatalf("Wrong return value (2) got: <%s>", ret2)
	}
}

// equivalent to basic.go
func TestLikeBasic(t *testing.T) {
	L := NewState()
	defer L.Close()
	L.OpenLibs()

	testCalled := 0

	test := func(L *State) int {
		testCalled++
		return 0
	}

	test2Arg := -1
	test2Argfrombottom := -1
	test2 := func(L *State) int {
		test2Arg = L.CheckInteger(-1)
		test2Argfrombottom = L.CheckInteger(1)
		return 0
	}

	L.GetGlobal("print")
	L.PushString("Hello World!")
	if err := L.Call(1, 0); err != nil {
		t.Fatalf("Call to print returned error")
	}

	L.PushGoFunction(test)
	L.PushGoFunction(test)
	L.PushGoFunction(test)
	L.PushGoFunction(test2)
	L.PushInteger(42)
	if err := L.Call(1, 0); err != nil {
		t.Fatalf("Call to print returned error")
	}
	if (test2Arg != 42) || (test2Argfrombottom != 42) {
		t.Fatalf("Call to test2 didn't work")
	}

	if err := L.Call(0, 0); err != nil {
		t.Fatalf("Call to print returned error")
	}
	if err := L.Call(0, 0); err != nil {
		t.Fatalf("Call to print returned error")
	}
	if err := L.Call(0, 0); err != nil {
		t.Fatalf("Call to print returned error")
	}
	if testCalled != 3 {
		t.Fatalf("Test function not called the correct number of times: %d\n", testCalled)
	}

	// this will fail as we didn't register test2 function
	if err := L.DoString("test2(42)"); err == nil {
		t.Fatal("No error when calling unregistered function")
	}
}

// equivalent to quickstart.go
func TestLikeQuickstart(t *testing.T) {
	adder := func(L *State) int {
		a := L.ToInteger(1)
		b := L.ToInteger(2)
		L.PushInteger(int64(a + b))
		return 1
	}

	L := NewState()
	defer L.Close()
	L.OpenLibs()

	L.Register("adder", adder)

	if err := L.DoString("return adder(2, 2)"); err != nil {
		t.Fatalf("Error during call to adder: %v\n", err)
	}
	if r := L.ToInteger(1); r != 4 {
		t.Fatalf("Wrong return value from adder (was: %d)\n", r)
	}
}

// equivalent to userdata.go
func TestLikeUserdata(t *testing.T) {
	type Userdata struct {
		a, b int
	}

	userDataProper := func(L *State) {
		rawptr := L.NewUserdata(uintptr(unsafe.Sizeof(Userdata{})))
		var ptr *Userdata
		ptr = (*Userdata)(rawptr)
		ptr.a = 2
		ptr.b = 3

		rawptr2 := L.ToUserdata(-1)
		ptr2 := (*Userdata)(rawptr2)

		if ptr != ptr2 {
			t.Fatalf("Failed to create userdata\n")
		}
	}

	testCalled := 0
	test := func(L *State) int {
		testCalled++
		return 0
	}

	goDefinedFunctions := func(L *State) {
		// example_function is registered inside Lua VM
		L.Register("test", test)

		// This code demonstrates checking that a value on the stack is a go function
		L.CheckStack(1)
		L.GetGlobal("test")
		if !L.IsGoFunction(-1) {
			t.Fatalf("IsGoFunction failed to recognize a Go function object")
		}
		L.Pop(1)

		// We call example_function from inside Lua VM
		testCalled = 0
		if err := L.DoString("test()"); err != nil {
			t.Fatalf("Error executing test function: %v\n", err)
		}
		if testCalled != 1 {
			t.Fatalf("It appears the test function wasn't actually called\n")
		}
	}

	type TestObject struct {
		AField int
	}

	goDefinedObjects := func(L *State) {
		z := &TestObject{42}

		L.PushGoStruct(z)
		L.SetGlobal("z")

		// This code demonstrates checking that a value on the stack is a go object
		L.CheckStack(1)
		L.GetGlobal("z")
		if !L.IsGoStruct(-1) {
			t.Fatal("IsGoStruct failed to recognize a Go struct\n")
		}
		L.Pop(1)

		// This code demonstrates access and assignment to a field of a go object
		if err := L.DoString("return z.AField"); err != nil {
			t.Fatal("Couldn't execute code")
		}
		before := L.ToInteger(-1)
		L.Pop(1)
		if before != 42 {
			t.Fatalf("Wrong value of z.AField before change (%d)\n", before)
		}
		if err := L.DoString("z.AField = 10;"); err != nil {
			t.Fatal("Couldn't execute code")
		}
		if err := L.DoString("return z.AField"); err != nil {
			t.Fatal("Couldn't execute code")
		}
		after := L.ToInteger(-1)
		L.Pop(1)
		if after != 10 {
			t.Fatalf("Wrong value of z.AField after change (%d)\n", after)
		}
	}

	L := NewState()
	defer L.Close()
	L.OpenLibs()

	userDataProper(L)
	goDefinedFunctions(L)
	goDefinedObjects(L)
}

func TestStackTrace(t *testing.T) {
	L := NewState()
	defer L.Close()
	L.OpenLibs()

	err := L.DoFile("../_example/calls.lua")
	if err == nil {
		t.Fatal("No error returned from the execution of calls.lua")
	}

	le := err.(*LuaError)

	if le.Code() != LUA_ERRERR {
		t.Fatalf("Wrong kind of error encountered running calls.lua: %v (%d %d)\n", le, le.Code(), LUA_ERRERR)
	}

	if len(le.StackTrace()) != 6 {
		t.Fatalf("Wrong size of stack trace (%v)\n", le.StackTrace())
	}
}

func TestConv(t *testing.T) {
	L := NewState()
	defer L.Close()
	L.OpenLibs()

	L.PushString("10")
	n := L.ToNumber(-1)
	if n != 10 {
		t.Fatalf("Wrong conversion (str -> int)")
	}
	if L.Type(-1) != LUA_TSTRING {
		t.Fatalf("Wrong type (str)")
	}

	L.Pop(1)

	L.PushInteger(10)
	s := L.ToString(-1)
	if s != "10" {
		t.Fatalf("Wrong conversion (int -> str)")
	}

	L.Pop(1)

	L.PushString("a\000test")
	s = L.ToString(-1)
	if s != "a\000test" {
		t.Fatalf("Wrong conversion (str -> str): <%s>", s)
	}
}

func TestDumpAndLoad(t *testing.T) {
	t.Skip("broken")
	L := NewState()
	defer L.Close()
	L.OpenLibs()

	loadret := L.LoadString(`print("msg from dump_and_load_test")`)
	if loadret != 0 {
		t.Fatalf("LoadString error: %v", loadret)
	}
	dumpret := L.Dump()
	if dumpret != 0 {
		t.Fatalf("Dump error: %v", dumpret)
	}

	isstring := L.IsString(-1)
	if !isstring {
		t.Fatalf("stack top not a string")
	}
	bytecodes := L.ToBytes(-1)
	loadret = L.Load(bytecodes, "chunk_from_dump_and_load_test")
	if loadret != 0 {
		t.Fatalf("Load error: %v", loadret)
	}
	err := L.Call(0, 0)
	if err != nil {
		t.Fatalf("Call error: %v", err)
	}
}

func TestCustomDebugHook(t *testing.T) {
	L := NewState()
	defer L.Close()

	L.SetHook(func(l *State) {
		l.RaiseError("stop")
	}, 1)

	err := L.DoString(`
		local x = 0
		while(1 ~= 0) do
			x = 2
		end
	`)

	if err == nil {
		t.Fatalf("Script should have raised an error")
	} else {
		if err.Error() != "stop" {
			t.Fatal("Error should be coming from the hook")
		}
	}
}

func TestSetUpValueShouldErrorOnInvalidCall(t *testing.T) {
	L := NewState()
	defer L.Close()

	L.PushGoFunction(func(L *State) int { return 0 })
	L.PushString("Hello")
	if L.SetUpvalue(-2, 1) == true { // Should not be succesful
		t.Fatal("SetUpvalue should return false because an upvalue can't be set on userdata")
	}
	if L.GetTop() != 2 {
		t.Fatal("Expected SetUpvalue to pop nothing on error")
	}
}

// This test is more an example to show why we can't directly set Upvalue on PushGoClosure, but rather
// have to rely on PushGoClosureWithUpvalues
func TestSetUpvalueOnGoClosureShouldErrorAt2(t *testing.T) {
	L := NewState()
	defer L.Close()

	L.PushGoClosure(func(L *State) int { return 0 })
	L.PushString("Hello")
	// Should be succesful because the function returned by PushGoClosure actually has an upvalue
	// at 1, which is the GoFunction userdata, that the closure calls.
	if !L.SetUpvalue(-2, 1) {
		t.Fatal("SetUpvalue should return false because an upvalue can't be set on userdata")
	}
	// Only the GoClosure should be left on the stack, since SetUpvalue was succesful
	if L.GetTop() != 1 {
		t.Fatalf("Expected GetTop to return 1 as SetUpvalue should pop value on success, but top is at: %d\n", L.GetTop())
	}

	L.PushString("Hello")
	// Since this closure only has 1 upvalue. It shouldn't be possible to add a second after the fact
	if L.SetUpvalue(-2, 2) {
		t.Fatal("SetUpvalue should return false because setting an upvalue at a higher index after creation is not possible")
	}
	// Since SetUpvalue failed, the string should remain
	if L.GetTop() != 2 {
		t.Fatal("Expected SetUpvalue to pop nothing on error")
	}
}

func TestPushGoClosureWithUpvalues(t *testing.T) {
	L := NewState()
	defer L.Close()

	closure := func(L *State) int {
		if !L.IsString(L.UpvalueIndex(2)) {
			t.Fatalf("upvalue 2 not set correctly, expected it to be a string")
		}
		if !L.IsNumber(L.UpvalueIndex(3)) {
			t.Fatalf("upvalue 3 not set correctly, expected it to be an number")
		}
		if L.ToString(L.UpvalueIndex(2)) != "Hello" {
			t.Fatalf("upvalue 2 not set correctly, expected it to equal \"Hello\"")
		}
		if L.ToInteger(L.UpvalueIndex(3)) != 15 {
			t.Fatalf("upvalue 3 not set correctly, expected it to equal 15")
		}
		return 0
	}

	L.PushString("Hello")
	L.PushInteger(15)
	L.PushGoClosureWithUpvalues(closure, 2)
	err := L.Call(0, 0)
	if err != nil {
		t.Fatalf("Call to function returned by PushGoClosureWithUpvalues should not fail: %s\n", err.Error())
	}
}
