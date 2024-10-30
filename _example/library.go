package main

import "github.com/aarzilli/golua/lua"

func test(L *lua.State) int {
	println("hello!")
	return 0
}

var funcs = map[string]lua.LuaGoFunction{
	"test": test,
}

const code = `
	local example = require("example")
	example.test()
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
