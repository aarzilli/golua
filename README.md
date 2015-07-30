Go Bindings for the lua C API
=========================

Simplest way to install:

	# go get -u github.com/aarzilli/golua/lua

Will work as long as your compiler can find a shared object called lua5.1 on linux, or lua anywhere else.
If your linux system uses "lua" as the shared object name for lua (for example, Fedora Core does this) you can install using:

	# go get -u -tags llua github.com/aarzilli/golua/lua


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

ON ERROR HANDLING
---------------------

Lua's exceptions are incompatible with Go, golua works around this incompatibility by setting up protected execution environments in `lua.State.DoString`, `lua.State.DoFile`  and lua.State.Call and turning every exception into a Go panic.

This means that:

1. In general you can't do any exception handling from Lua, `pcall` and `xpcall` are renamed to `unsafe_pcall` and `unsafe_xpcall`. They are only safe to be called from Lua code that never calls back to Go. Use at your own risk.

2. The call to lua.State.Error, present in previous versions of this library, has been removed as it is nonsensical

3. Method calls on a newly created `lua.State` happen in an unprotected environment, if Lua throws an exception as a result your program will be terminated. If this is undesirable perform your initialization like this:

```go
func LuaStateInit(L *lua.State) int {
	… initialization goes here…
	return 0
}

…
L.PushGoFunction(LuaStateInit)
err := L.Call(0, 0)
…
```

ON THREADS AND COROUTINES
---------------------

'lua.State' is not thread safe, but the library itself is. Lua's coroutines exist but (to my knowledge) have never been tested and are likely to encounter the same problems that errors have, use at your own peril.

ODDS AND ENDS
---------------------

* Support for lua 5.2 is in the lua5.2 branch, this branch only supports lua5.1.
* Compiling from source yields only a static link library (liblua.a), you can either produce the dynamic link library on your own or use the `luaa` build tag.

LUAJIT
---------------------

To link with [luajit-2.0.x](http://luajit.org/luajit.html), you can use CGO_CFLAGS and CGO_LDFLAGS environment variables

```
$ CGO_CFLAGS=`pkg-config luajit --cflags`
$ CGO_LDFLAGS=`pkg-config luajit --libs`
$ go get -f -u github.com/aarzilli/golua/lua
```

CONTRIBUTORS
---------------------

* Adam Fitzgerald (original author)
* Alessandro Arzilli
* Steve Donovan
* Harley Laue
* James Nurmi
* Ruitao
* Xushiwei
* Isaint
* hsinhoyeh
* Viktor Palmkvist
* HongZhen Peng
* Admin36

SEE ALSO
---------------------

[Luar](https://github.com/stevedonovan/luar/) is a reflection layer on top of golua API providing a simplified way to publish go functions to a Lua VM.

Licensing
-------------
GoLua is released under the MIT license.
Please see the LICENSE file for more information.

Lua is Copyright (c) Lua.org, PUC-Rio.  All rights reserved.
