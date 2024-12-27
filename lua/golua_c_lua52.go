//go:build lua52
// +build lua52

package lua

/*
#include <lua.h>
#include <lauxlib.h>
#include <lualib.h>
#include <stdlib.h>

typedef struct _chunk {
	int size; // chunk size
	char *buffer; // chunk data
	char* toread; // chunk to read
} chunk;

static const char * reader (lua_State *L, void *ud, size_t *sz) {
	chunk *ck = (chunk *)ud;
	if (ck->size > LUAL_BUFFERSIZE) {
		ck->size -= LUAL_BUFFERSIZE;
		*sz = LUAL_BUFFERSIZE;
		ck->toread = ck->buffer;
		ck->buffer += LUAL_BUFFERSIZE;
	}else{
		*sz = ck->size;
		ck->toread = ck->buffer;
		ck->size = 0;
	}
	return ck->toread;
}

static int writer (lua_State *L, const void* b, size_t size, void* B) {
	static int count=0;
	(void)L;
	luaL_addlstring((luaL_Buffer*) B, (const char *)b, size);
	return 0;
}

// load function chunk dumped from dump_chunk
int load_chunk(lua_State *L, char *b, int size, const char* chunk_name) {
	chunk ck;
	ck.buffer = b;
	ck.size = size;
	int err;
	err = lua_load(L, reader, &ck, chunk_name, NULL);
	if (err != 0) {
		return luaL_error(L, "unable to load chunk, err: %d", err);
	}
	return 0;
}

void clua_openio(lua_State* L)
{
	luaL_requiref(L, "io", &luaopen_io, 1);
	lua_pop(L, 1);
}

void clua_openmath(lua_State* L)
{
	luaL_requiref(L, "math", &luaopen_math, 1);
	lua_pop(L, 1);
}

void clua_openpackage(lua_State* L)
{
	luaL_requiref(L, "package", &luaopen_package, 1);
	lua_pop(L, 1);
}

void clua_openstring(lua_State* L)
{
	luaL_requiref(L, "string", &luaopen_string, 1);
	lua_pop(L, 1);
}

void clua_opentable(lua_State* L)
{
	luaL_requiref(L, "table", &luaopen_table, 1);
	lua_pop(L, 1);
}

void clua_openos(lua_State* L)
{
	luaL_requiref(L, "os", &luaopen_os, 1);
	lua_pop(L, 1);
}

void clua_opencoroutine(lua_State *L)
{
	luaL_requiref(L, "coroutine", &luaopen_coroutine, 1);
	lua_pop(L, 1);
}

void clua_opendebug(lua_State *L)
{
	luaL_requiref(L, "debug", &luaopen_debug, 1);
	lua_pop(L, 1);
}

void clua_openbit32(lua_State *L)
{
	luaL_requiref(L, "bit32", &luaopen_bit32, 1);
	lua_pop(L, 1);
}

// dump function chunk from luaL_loadstring
int dump_chunk (lua_State *L) {
	luaL_Buffer b;
	luaL_checktype(L, -1, LUA_TFUNCTION);
	lua_settop(L, -1);
	luaL_buffinit(L,&b);
	int err;
	err = lua_dump(L, writer, &b);
	if (err != 0){
	return luaL_error(L, "unable to dump given function, err:%d", err);
	}
	luaL_pushresult(&b);
	return 0;
}
*/
import "C"

import "unsafe"

func luaToInteger(s *C.lua_State, n C.int) C.long {
	return C.lua_tointegerx(s, n, nil)
}

func luaToNumber(s *C.lua_State, n C.int) C.double {
	return C.lua_tonumberx(s, n, nil)
}

func lualLoadFile(s *C.lua_State, filename *C.char) C.int {
	return C.luaL_loadfilex(s, filename, nil)
}

// lua_equal
func (L *State) Equal(index1, index2 int) bool {
	return C.lua_compare(L.s, C.int(index1), C.int(index2), C.LUA_OPEQ) == 1
}

// lua_lessthan
func (L *State) LessThan(index1, index2 int) bool {
	return C.lua_compare(L.s, C.int(index1), C.int(index2), C.LUA_OPLT) == 1
}

func (L *State) ObjLen(index int) uint {
	return uint(C.lua_rawlen(L.s, C.int(index)))
}

// [lua_tointeger] -> [-0, +0, –]
//
// Equivalent to lua_tointegerx with isnum equal to NULL.
//
// [lua_tointeger]: https://www.lua.org/manual/5.2/manual.html#lua_tointeger
func (L *State) ToInteger(index int) int {
	return int(C.lua_tointegerx(L.s, C.int(index), nil))
}

// [lua_tonumber] -> [-0, +0, –]
//
// Equivalent to lua_tonumberx with isnum equal to NULL.
//
// [lua_tonumber]: https://www.lua.org/manual/5.2/manual.html#lua_tonumber
func (L *State) ToNumber(index int) float64 {
	return float64(C.lua_tonumberx(L.s, C.int(index), nil))
}

// [lua_yield] -> [-?, +?, –]
//
// This function is equivalent to lua_yieldk, but it has no continuation (see §4.7). Therefore, when the thread resumes, it returns to the function that called the function calling lua_yield.
//
// [lua_yield]: https://www.lua.org/manual/5.2/manual.html#lua_yield
func (L *State) Yield(nresults int) int {
	return int(C.lua_yieldk(L.s, C.int(nresults), 0, nil))
}

func (L *State) pcall(nargs, nresults, errfunc int) int {
	return int(C.lua_pcallk(L.s, C.int(nargs), C.int(nresults), C.int(errfunc), 0, nil))
}

// Pushes on the stack the value of a global variable (lua_getglobal)
func (L *State) GetGlobal(name string) {
	Ck := C.CString(name)
	defer C.free(unsafe.Pointer(Ck))
	C.lua_getglobal(L.s, Ck)
}

// [lua_resume] -> [-?, +?, –]
//
// Starts and resumes a coroutine in a given thread.
//
// [lua_resume]: https://www.lua.org/manual/5.2/manual.html#lua_resume
func (L *State) Resume(narg int) int {
	return int(C.lua_resume(L.s, nil, C.int(narg)))
}

// [lua_setglobal] -> [-1, +0, e]
//
// Pops a value from the stack and sets it as the new value of global name.
//
// [lua_setglobal]: https://www.lua.org/manual/5.2/manual.html#lua_setglobal
func (L *State) SetGlobal(name string) {
	Cname := C.CString(name)
	defer C.free(unsafe.Pointer(Cname))
	C.lua_setglobal(L.s, Cname)
}

// Calls luaopen_debug
func (L *State) OpenDebug() {
	C.clua_opendebug(L.s)
}

// Calls luaopen_bit32
func (L *State) OpenBit32() {
	C.clua_openbit32(L.s)
}

// Calls luaopen_coroutine
func (L *State) OpenCoroutine() {
	C.clua_opencoroutine(L.s)
}

// [lua_insert] -> [-1, +1, –]
//
// Moves the top element into the given valid index, shifting up the elements above this index to open space. This function cannot be called with a pseudo-index, because a pseudo-index is not an actual stack position.
//
// [lua_insert]: https://www.lua.org/manual/5.2/manual.html#lua_insert
func (L *State) Insert(index int) { C.lua_insert(L.s, C.int(index)) }

// [lua_remove] -> [-1, +0, –]
//
// Removes the element at the given valid index, shifting down the elements above this index to fill the gap. This function cannot be called with a pseudo-index, because a pseudo-index is not an actual stack position.
//
// [lua_remove]: https://www.lua.org/manual/5.2/manual.html#lua_remove
func (L *State) Remove(index int) {
	C.lua_remove(L.s, C.int(index))
}

// [lua_replace] -> [-1, +0, –]
//
// Moves the top element into the given valid index without shifting any element (therefore replacing the value at the given index), and then pops the top element.
//
// [lua_replace]: https://www.lua.org/manual/5.2/manual.html#lua_replace
func (L *State) Replace(index int) {
	C.lua_replace(L.s, C.int(index))
}

// [lua_rawgeti] -> [-0, +1, –]
//
// Pushes onto the stack the value t[n], where t is the table at the given index. The access is raw; that is, it does not invoke metamethods.
//
// [lua_rawgeti]: https://www.lua.org/manual/5.2/manual.html#lua_rawgeti
func (L *State) RawGeti(index int, n int) {
	C.lua_rawgeti(L.s, C.int(index), C.int(n))
}

// [lua_rawseti] -> [-1, +0, e]
//
// Does the equivalent of t[n] = v, where t is the table at the given index and v is the value at the top of the stack.
//
// [lua_rawseti]: https://www.lua.org/manual/5.2/manual.html#lua_rawseti
func (L *State) RawSeti(index int, n int) {
	C.lua_rawseti(L.s, C.int(index), C.int(n))
}

// [lua_gc] -> [-0, +0, e]
//
// Controls the garbage collector.
//
// [lua_gc]: https://www.lua.org/manual/5.2/manual.html#lua_gc
func (L *State) GC(what, data int) int {
	return int(C.lua_gc(L.s, C.int(what), C.int(data)))
}
