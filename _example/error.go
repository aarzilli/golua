package main

import (
	"github.com/aarzilli/golua/lua"
	"fmt"
	"errors"
	"os"
)

func testDefault(L *lua.State) {
	err := L.DoString("print(\"Unknown variable\" .. x)")
	fmt.Printf("Error is: %v\n", err)
	if err == nil {
		fmt.Printf("Error shouldn't have been nil\n")
		os.Exit(1)
	}
}

func faultyfunc(L *lua.State) int {
	panic(errors.New("An error"))
}

func testRegistered(L *lua.State) {
	L.Register("faultyfunc", faultyfunc)
	err := L.DoString("faultyfunc()")
	fmt.Printf("Error is %v\n", err)
	if err == nil {
		fmt.Printf("Error shouldn't have been nil\n")
		os.Exit(1)
	}
}

func test2(L *lua.State) {
	err := L.DoString("error(\"Some error\")")
	fmt.Printf("Error is %v\n", err)
	if err == nil {
		fmt.Printf("Error shouldn't have been nil\n")
		os.Exit(1)
	}
}

func main() {
	L := lua.NewState()
	defer L.Close()
	L.OpenLibs()

	testDefault(L)
	testRegistered(L)
	test2(L)
}
