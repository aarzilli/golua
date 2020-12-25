Go Bindings for the lua C API
=========================

[![Build Status](https://travis-ci.org/aarzilli/golua.svg?branch=master)](https://travis-ci.org/aarzilli/golua)

Simplest way to install:

	# go get github.com/aarzilli/golua/lua

You can then try to run the examples:

	$ cd golua/_example/
	$ go run basic.go
	$ go run alloc.go
	$ go run panic.go
	$ go run userdata.go

This library is configured using build tags. By default it will look for a library (or "shared object") called:

* lua5.1 on Linux and macOS
* lua on Windows
* lua-5.1 on FreeBSD

If this doesn't work `-tags luadash5.1` can be used to force `lua-5.1`, and `-tags llua` can be used to force `lua`.

If you want to statically link to liblua.a you can do that with `-tags luaa`. Luajit can also be used by
specifying `-tags luajit`.

The library uses lua5.1 by default but also supports lua5.2 by specifying `-tags lua52`, lua5.3 by
specifying `-tags lua53`, and lua5.4 by specifying `-tags lua54`.

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
L.GetGlobal("print")
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

* If you want to build against lua5.2, lua5.3, or lua5.4 use the build tags lua52, lua53, or lua54 respectively.
* Compiling from source yields only a static link library (liblua.a), you can either produce the dynamic link library on your own or use the `luaa` build tag.

LUAJIT
---------------------

To link with [luajit-2.0.x](http://luajit.org/luajit.html), you can use CGO_CFLAGS and CGO_LDFLAGS environment variables

```
$ CGO_CFLAGS=`pkg-config luajit --cflags`
$ CGO_LDFLAGS=`pkg-config luajit --libs-only-L`
$ go get -f -u -tags luajit github.com/aarzilli/golua/lua
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
* Pierre Neidhardt (@Ambrevar)
* HuangWei (@huangwei1024)
* Adam Saponara

SEE ALSO
---------------------

- [Luar](https://github.com/stevedonovan/luar/) is a reflection layer on top of golua API providing a simplified way to publish go functions to a Lua VM.
- [lunatico](https://github.com/fiatjaf/lunatico) is a reflection layer that allows you to push and read Go values to a Lua VM without understanding the Lua stack.
- [Golua unicode](https://github.com/Ambrevar/golua) is an extension library that adds unicode support to golua and replaces lua regular expressions with re2.

Licensing
-------------
GoLua is released under the MIT license.
Please see the LICENSE file for more information.

Lua is Copyright (c) Lua.org, PUC-Rio.  All rights reserved.
