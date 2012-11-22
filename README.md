Go Bindings for the lua C API
=========================

Simplest way to install:

	# go get -u github.com/aarzilli/golua/lua

Will work as long as `pkg-config` knows about lua.

You can then try to run the examples:

	$ cd /usr/local/go/src/pkg/github.com/aarzilli/golua/example/
	$ go run basic.go
	$ go run alloc.go
	$ go run panic.go
	$ go run userdata.go

QUICK START
---------------------

Create a new Virtual Machine with:

```go
L := lua.NewState()
L.OpenLibs()
defer L.Close()
```

Lua's Virtual Machine is stack based, you can call lua functions like this:

```go
// push "print" function on the stack
L.GetField(lua.LUA_GLOBALSINDEX, "print")
// push the string "Hello World!" on the stack
L.PushString("Hello World!")
// call print with one argument, expecting no results
L.Call(1, 0)
```

Of course this isn't very useful, more useful is executing lua code from a file or from a string:

```go
// executes a string of lua code
err := L.DoString("...")
// executes a file
err = L.DoFile(filename)
```

You will also probably want to publish go functions to the virtual machine, you can do it by:

```go
func adder(L *lua.State) int {
	a := L.ToInteger(1)
	b := L.ToInteger(2)
	L.PushInteger(a + b)
	return 1 // number of return values
}

func main() {
	L := lua.NewState()
	defer L.Close()
	L.OpenLibs()

	L.Register("adder", adder)
	L.DoString("print(adder(2, 2))")
}
```

Licensing
-------------
GoLua is released under the MIT license.
Please see the LICENSE file for more information.

Lua is Copyright (c) Lua.org, PUC-Rio.  All rights reserved.



