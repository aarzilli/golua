package main

import "github.com/aarzilli/golua/lua"

func test(L *lua.State) int {
	println("hello!")
	return 0
}

func test2(L *lua.State) int {
	println("world!")
	return 0
}

var funcs = map[string]lua.LuaGoFunction{
	"test": test,
	"test2": test2,
}

const code = `
	local example = require("example")
	example.test()
	example.test2()
	`

func main() {
	L := lua.NewState()
	defer L.Close()
	L.OpenLibs()
	
	L.RegisterLib("example", funcs)

	if err := L.DoString(code); err != nil {
		panic(err)
	}
}
