package main

import (
	"github.com/aarzilli/golua/lua"
	"unsafe"
	"fmt"
)

type Userdata struct {
	a, b int
}

func userDataProper(L *lua.State) {
	rawptr := L.NewUserdata(uintptr(unsafe.Sizeof(Userdata{})))
	var ptr *Userdata
	ptr = (*Userdata)(rawptr)
	ptr.a = 2
	ptr.b = 3

	fmt.Println(ptr)

	rawptr2 := L.ToUserdata(-1)
	ptr2 := (*Userdata)(rawptr2)

	fmt.Println(ptr2)
}

func example_function(L *lua.State) int {
	fmt.Println("Heeeeelllllooooooooooo nuuurse!!!!")
	return 0
}

func goDefinedFunctions(L *lua.State) {
	/* example_function is registered inside Lua VM */
	L.Register("example_function", example_function)

	/* This code demonstrates checking that a value on the stack is a go function */
	L.CheckStack(1)
	L.GetGlobal("example_function")
	if !L.IsGoFunction(-1) {
		panic("Not a go function")
	}
	L.Pop(1)

	/* We call example_function from inside Lua VM */
	L.MustDoString("example_function()")
}

type TestObject struct {
	AField int
}

func goDefinedObjects(L *lua.State) {
	t := &TestObject{42}

	L.PushGoStruct(t)
	L.SetGlobal("t")

	/* This code demonstrates checking that a value on the stack is a go object */
	L.CheckStack(1)
	L.GetGlobal("t")
	if !L.IsGoStruct(-1) {
		panic("Not a go struct")
	}
	L.Pop(1)

	/* This code demonstrates access and assignment to a field of a go object */
	L.MustDoString("print('AField of t is: ' .. t.AField .. ' before assignment');")
	L.MustDoString("t.AField = 10;")
	L.MustDoString("print('AField of t is: ' .. t.AField .. ' after assignment');")
}

func main() {
	L := lua.NewState()
	defer L.Close()
	L.OpenLibs()

	/*
		This function stores a go object inside Lua VM
	*/
	userDataProper(L)

	/*
		This function demonstrates exposing a function implemented in go to interpreted Lua code
	*/
	goDefinedFunctions(L)

	goDefinedObjects(L)
}
