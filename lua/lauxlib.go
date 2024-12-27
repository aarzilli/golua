package lua

//#include <lua.h>
//#include <lauxlib.h>
//#include <lualib.h>
//#include <stdlib.h>
//#include "golua.h"
import "C"

import (
	"context"
	"unsafe"
)

type LuaError struct {
	code       int
	message    string
	stackTrace []LuaStackEntry
}

func (err *LuaError) Error() string {
	return err.message
}

func (err *LuaError) Code() int {
	return err.code
}

func (err *LuaError) StackTrace() []LuaStackEntry {
	return err.stackTrace
}

// [luaL_argcheck] -> [-0, +0, v]
//
// Checks whether cond is true. If not, raises an error with the following message, where func is retrieved from the call stack:
//
// [luaL_argcheck]: https://www.lua.org/manual/5.1/manual.html#lual_argcheck
// WARNING: before b30b2c62c6712c6683a9d22ff0abfa54c8267863 the function ArgCheck had the opposite behaviour
func (L *State) Argcheck(cond bool, narg int, extramsg string) {
	if !cond {
		Cextramsg := C.CString(extramsg)
		defer C.free(unsafe.Pointer(Cextramsg))
		C.luaL_argerror(L.s, C.int(narg), Cextramsg)
	}
}

// [luaL_argerror] -> [-0, +0, v]
//
// Raises an error with the following message, where func is retrieved from the call stack:
//
// [luaL_argerror]: https://www.lua.org/manual/5.1/manual.html#lual_argerror
func (L *State) ArgError(narg int, extramsg string) int {
	Cextramsg := C.CString(extramsg)
	defer C.free(unsafe.Pointer(Cextramsg))
	return int(C.luaL_argerror(L.s, C.int(narg), Cextramsg))
}

// [luaL_callmeta] -> [-0, +(0|1), e]
//
// Calls a metamethod.
//
// [luaL_callmeta]: https://www.lua.org/manual/5.1/manual.html#lual_callmeta
func (L *State) CallMeta(obj int, e string) int {
	Ce := C.CString(e)
	defer C.free(unsafe.Pointer(Ce))
	return int(C.luaL_callmeta(L.s, C.int(obj), Ce))
}

// [luaL_checkany] -> [-0, +0, v]
//
// Checks whether the function has an argument of any type (including nil) at position narg.
//
// [luaL_checkany]: https://www.lua.org/manual/5.1/manual.html#lual_checkany
func (L *State) CheckAny(narg int) {
	C.luaL_checkany(L.s, C.int(narg))
}

// [luaL_checkinteger] -> [-0, +0, v]
//
// Checks whether the function argument narg is a number and returns this number cast to a lua_Integer.
//
// [luaL_checkinteger]: https://www.lua.org/manual/5.1/manual.html#lual_checkinteger
func (L *State) CheckInteger(narg int) int {
	return int(C.luaL_checkinteger(L.s, C.int(narg)))
}

// [luaL_checknumber] -> [-0, +0, v]
//
// Checks whether the function argument narg is a number and returns this number.
//
// [luaL_checknumber]: https://www.lua.org/manual/5.1/manual.html#lual_checknumber
func (L *State) CheckNumber(narg int) float64 {
	return float64(C.luaL_checknumber(L.s, C.int(narg)))
}

// [luaL_checkstring] -> [-0, +0, v]
//
// Checks whether the function argument narg is a string and returns this string.
//
// [luaL_checkstring]: https://www.lua.org/manual/5.1/manual.html#lual_checkstring
func (L *State) CheckString(narg int) string {
	var length C.size_t
	return C.GoString(C.luaL_checklstring(L.s, C.int(narg), &length))
}

// [luaL_checkoption] -> [-0, +0, v]
//
// Checks whether the function argument narg is a string and searches for this string in the array lst (which must be NULL-terminated). Returns the index in the array where the string was found. Raises an error if the argument is not a string or if the string cannot be found.
//
// [luaL_checkoption]: https://www.lua.org/manual/5.1/manual.html#lual_checkoption
//
// BUG(everyone_involved): not implemented
func (L *State) CheckOption(narg int, def string, lst []string) int {
	// TODO: complication: lst conversion to const char* lst[] from string slice
	return 0
}

// [luaL_checktype] -> [-0, +0, v]
//
// Checks whether the function argument narg has type t. See lua_type for the encoding of types for t.
//
// [luaL_checktype]: https://www.lua.org/manual/5.1/manual.html#lual_checktype
func (L *State) CheckType(narg int, t LuaValType) {
	C.luaL_checktype(L.s, C.int(narg), C.int(t))
}

// [luaL_checkudata] -> [-0, +0, v]
//
// Checks whether the function argument narg is a userdata of the type tname (see luaL_newmetatable).
//
// [luaL_checkudata]: https://www.lua.org/manual/5.1/manual.html#lual_checkudata
func (L *State) CheckUdata(narg int, tname string) unsafe.Pointer {
	Ctname := C.CString(tname)
	defer C.free(unsafe.Pointer(Ctname))
	return unsafe.Pointer(C.luaL_checkudata(L.s, C.int(narg), Ctname))
}

// Executes file, returns nil for no errors or the lua error string on failure
func (L *State) DoFile(filename string) error {
	if r := L.LoadFile(filename); r != 0 {
		return &LuaError{r, L.ToString(-1), L.StackTrace()}
	}
	return L.Call(0, LUA_MULTRET)
}

// Executes the string, returns nil for no errors or the lua error string on failure
func (L *State) DoString(str string) error {
	if r := L.LoadString(str); r != 0 {
		return &LuaError{r, L.ToString(-1), L.StackTrace()}
	}
	return L.Call(0, LUA_MULTRET)
}

// Like DoString but panics on error
func (L *State) MustDoString(str string) {
	if err := L.DoString(str); err != nil {
		panic(err)
	}
}

// [luaL_getmetafield] -> [-0, +(0|1), m]
//
// Pushes onto the stack the field e from the metatable of the object at index obj. If the object does not have a metatable, or if the metatable does not have this field, returns 0 and pushes nothing.
//
// [luaL_getmetafield]: https://www.lua.org/manual/5.1/manual.html#lual_getmetafield
func (L *State) GetMetaField(obj int, e string) bool {
	Ce := C.CString(e)
	defer C.free(unsafe.Pointer(Ce))
	return C.luaL_getmetafield(L.s, C.int(obj), Ce) != 0
}

// [luaL_getmetatable] -> [-0, +1, -]
//
// Pushes onto the stack the metatable associated with name tname in the registry (see luaL_newmetatable).
//
// [luaL_getmetatable]: https://www.lua.org/manual/5.1/manual.html#lual_getmetatable
func (L *State) LGetMetaTable(tname string) {
	Ctname := C.CString(tname)
	defer C.free(unsafe.Pointer(Ctname))
	C.lua_getfield(L.s, LUA_REGISTRYINDEX, Ctname)
}

// [luaL_gsub] -> [-0, +1, m]
//
// Creates a copy of string s by replacing any occurrence of the string p with the string r. Pushes the resulting string on the stack and returns it.
//
// [luaL_gsub]: https://www.lua.org/manual/5.1/manual.html#lual_gsub
func (L *State) GSub(s string, p string, r string) string {
	Cs := C.CString(s)
	Cp := C.CString(p)
	Cr := C.CString(r)
	defer func() {
		C.free(unsafe.Pointer(Cs))
		C.free(unsafe.Pointer(Cp))
		C.free(unsafe.Pointer(Cr))
	}()

	return C.GoString(C.luaL_gsub(L.s, Cs, Cp, Cr))
}

// [luaL_loadfile] -> [-0, +1, m]
//
// Loads a file as a Lua chunk. This function uses lua_load to load the chunk in the file named filename. If filename is NULL, then it loads from the standard input. The first line in the file is ignored if it starts with a #.
//
// [luaL_loadfile]: https://www.lua.org/manual/5.1/manual.html#lual_loadfile
func (L *State) LoadFile(filename string) int {
	Cfilename := C.CString(filename)
	defer C.free(unsafe.Pointer(Cfilename))
	return int(lualLoadFile(L.s, Cfilename))
}

// [luaL_loadstring] -> [-0, +1, m]
//
// Loads a string as a Lua chunk. This function uses lua_load to load the chunk in the zero-terminated string s.
//
// [luaL_loadstring]: https://www.lua.org/manual/5.1/manual.html#lual_loadstring
func (L *State) LoadString(s string) int {
	Cs := C.CString(s)
	defer C.free(unsafe.Pointer(Cs))
	return int(C.luaL_loadstring(L.s, Cs))
}

// [lua_dump] -> [-0, +0, m]
//
// Dumps a function as a binary chunk. Receives a Lua function on the top of the stack and produces a binary chunk that, if loaded again, results in a function equivalent to the one dumped. As it produces parts of the chunk, lua_dump calls function writer (see lua_Writer) with the given data to write them.
//
// [lua_dump]: https://www.lua.org/manual/5.1/manual.html#lua_dump
func (L *State) Dump() int {
	ret := int(C.dump_chunk(L.s))
	return ret
}

// [lua_load] -> [-0, +1, -]
//
// Loads a Lua chunk. If there are no errors, lua_load pushes the compiled chunk as a Lua function on top of the stack. Otherwise, it pushes an error message. The return values of lua_load are:
//
// [lua_load]: https://www.lua.org/manual/5.1/manual.html#lua_load
func (L *State) Load(bs []byte, name string) int {
	chunk := C.CString(string(bs))
	ckname := C.CString(name)
	defer C.free(unsafe.Pointer(chunk))
	defer C.free(unsafe.Pointer(ckname))
	ret := int(C.load_chunk(L.s, chunk, C.int(len(bs)), ckname))
	if ret != 0 {
		return ret
	}
	return 0
}

// [luaL_newmetatable] -> [-0, +1, m]
//
// If the registry already has the key tname, returns 0. Otherwise, creates a new table to be used as a metatable for userdata, adds it to the registry with key tname, and returns 1.
//
// [luaL_newmetatable]: https://www.lua.org/manual/5.1/manual.html#lual_newmetatable
func (L *State) NewMetaTable(tname string) bool {
	Ctname := C.CString(tname)
	defer C.free(unsafe.Pointer(Ctname))
	return C.luaL_newmetatable(L.s, Ctname) != 0
}

// [luaL_newstate] -> [-0, +0, -]
//
// Creates a new Lua state. It calls lua_newstate with an allocator based on the standard C realloc function and then sets a panic function (see lua_atpanic) that prints an error message to the standard error output in case of fatal errors.
//
// [luaL_newstate]: https://www.lua.org/manual/5.1/manual.html#lual_newstate
func NewState() *State {
	ls := (C.luaL_newstate())
	if ls == nil {
		return nil
	}
	L := newState(ls)
	return L
}

func NewStateWithContext(ctx context.Context) *State {
	L := NewState()
	L.ctx = ctx
	return L
}

func (L *State) Context() context.Context {
	return L.ctx
}

// [luaL_openlibs] -> [-0, +0, m]
//
// Opens all standard Lua libraries into the given state.
//
// [luaL_openlibs]: https://www.lua.org/manual/5.1/manual.html#lual_openlibs
func (L *State) OpenLibs() {
	C.luaL_openlibs(L.s)
	C.clua_hide_pcall(L.s)
}

// [luaL_optinteger] -> [-0, +0, v]
//
// If the function argument narg is a number, returns this number cast to a lua_Integer. If this argument is absent or is nil, returns d. Otherwise, raises an error.
//
// [luaL_optinteger]: https://www.lua.org/manual/5.1/manual.html#lual_optinteger
func (L *State) OptInteger(narg int, d int) int {
	return int(C.luaL_optinteger(L.s, C.int(narg), C.lua_Integer(d)))
}

// [luaL_optnumber] -> [-0, +0, v]
//
// If the function argument narg is a number, returns this number. If this argument is absent or is nil, returns d. Otherwise, raises an error.
//
// [luaL_optnumber]: https://www.lua.org/manual/5.1/manual.html#lual_optnumber
func (L *State) OptNumber(narg int, d float64) float64 {
	return float64(C.luaL_optnumber(L.s, C.int(narg), C.lua_Number(d)))
}

// [luaL_optstring] -> [-0, +0, v]
//
// If the function argument narg is a string, returns this string. If this argument is absent or is nil, returns d. Otherwise, raises an error.
//
// [luaL_optstring]: https://www.lua.org/manual/5.1/manual.html#lual_optstring
func (L *State) OptString(narg int, d string) string {
	var length C.size_t
	Cd := C.CString(d)
	defer C.free(unsafe.Pointer(Cd))
	return C.GoString(C.luaL_optlstring(L.s, C.int(narg), Cd, &length))
}

// [luaL_ref] -> [-1, +0, m]
//
// Creates and returns a reference, in the table at index t, for the object at the top of the stack (and pops the object).
//
// [luaL_ref]: https://www.lua.org/manual/5.1/manual.html#lual_ref
func (L *State) Ref(t int) int {
	return int(C.luaL_ref(L.s, C.int(t)))
}

// [luaL_typename] -> [-0, +0, -]
//
// Returns the name of the type of the value at the given index.
//
// [luaL_typename]: https://www.lua.org/manual/5.1/manual.html#lual_typename
func (L *State) LTypename(index int) string {
	return C.GoString(C.lua_typename(L.s, C.lua_type(L.s, C.int(index))))
}

// [luaL_unref] -> [-0, +0, -]
//
// Releases reference ref from the table at index t (see luaL_ref). The entry is removed from the table, so that the referred object can be collected. The reference ref is also freed to be used again.
//
// [luaL_unref]: https://www.lua.org/manual/5.1/manual.html#lual_unref
func (L *State) Unref(t int, ref int) {
	C.luaL_unref(L.s, C.int(t), C.int(ref))
}

// [luaL_where] -> [-0, +1, m]
//
// Pushes onto the stack a string identifying the current position of the control at level lvl in the call stack. Typically this string has the following format:
//
// [luaL_where]: https://www.lua.org/manual/5.1/manual.html#lual_where
func (L *State) Where(lvl int) {
	C.luaL_where(L.s, C.int(lvl))
}
