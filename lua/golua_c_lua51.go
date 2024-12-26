//go:build !lua52 && !lua53 && !lua54
// +build !lua52,!lua53,!lua54

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
	err = lua_load(L, reader, &ck, chunk_name);
	if (err != 0) {
		return luaL_error(L, "unable to load chunk, err: %d", err);
	}
	return 0;
}

void clua_openio(lua_State* L)
{
	lua_pushcfunction(L,&luaopen_io);
	lua_pushstring(L,"io");
	lua_call(L, 1, 0);
}

void clua_openmath(lua_State* L)
{
	lua_pushcfunction(L,&luaopen_math);
	lua_pushstring(L,"math");
	lua_call(L, 1, 0);
}

void clua_openpackage(lua_State* L)
{
	lua_pushcfunction(L,&luaopen_package);
	lua_pushstring(L,"package");
	lua_call(L, 1, 0);
}

void clua_openstring(lua_State* L)
{
	lua_pushcfunction(L,&luaopen_string);
	lua_pushstring(L,"string");
	lua_call(L, 1, 0);
}

void clua_opentable(lua_State* L)
{
	lua_pushcfunction(L,&luaopen_table);
	lua_pushstring(L,"table");
	lua_call(L, 1, 0);
}

void clua_openos(lua_State* L)
{
	lua_pushcfunction(L,&luaopen_os);
	lua_pushstring(L,"os");
	lua_call(L, 1, 0);
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
	return C.lua_tointeger(s, n)
}

func luaToNumber(s *C.lua_State, n C.int) C.double {
	return C.lua_tonumber(s, n)
}

func lualLoadFile(s *C.lua_State, filename *C.char) C.int {
	return C.luaL_loadfile(s, filename)
}

// lua_equal
/*
 * [-0, +0, e]
 * Returns 1 if the two values in acceptable indices index1 and index2 are equal, following the semantics of the Lua == operator (that is, may call metamethods). Otherwise returns 0. Also returns 0 if any of the indices is non valid.
 */
func (L *State) Equal(index1, index2 int) bool {
	return C.lua_equal(L.s, C.int(index1), C.int(index2)) == 1
}

// lua_getfenv
/*
 * [-0, +1, -]
 * Pushes onto the stack the environment table of the value at the given index.
 */
func (L *State) GetfEnv(index int) {
	C.lua_getfenv(L.s, C.int(index))
}

// lua_lessthan
/*
 * [-0, +0, e]
 * Returns 1 if the value at acceptable index index1 is smaller than the value at acceptable index index2, following the semantics of the Lua < operator (that is, may call metamethods). Otherwise returns 0. Also returns 0 if any of the indices is non valid.
 */
func (L *State) LessThan(index1, index2 int) bool {
	return C.lua_lessthan(L.s, C.int(index1), C.int(index2)) == 1
}

// lua_setfenv
/*
 * [-1, +0, -]
 * Pops a table from the stack and sets it as the new environment for the value at the given index. If the value at the given index is neither a function nor a thread nor a userdata, lua_setfenv returns 0. Otherwise it returns 1.
 */
func (L *State) SetfEnv(index int) {
	C.lua_setfenv(L.s, C.int(index))
}

func (L *State) ObjLen(index int) uint {
	return uint(C.lua_objlen(L.s, C.int(index)))
}

// lua_tointeger
/*
 * [-0, +0, -]
 * Converts the Lua value at the given acceptable index to the signed integral type lua_Integer. The Lua value must be a number or a string convertible to a number (see §2.2.1); otherwise, lua_tointeger returns 0.
 */
func (L *State) ToInteger(index int) int {
	return int(C.lua_tointeger(L.s, C.int(index)))
}

// lua_tonumber
/*
 * [-0, +0, -]
 * Converts the Lua value at the given acceptable index to the C type lua_Number (see lua_Number). The Lua value must be a number or a string convertible to a number (see §2.2.1); otherwise, lua_tonumber returns 0.
 */
func (L *State) ToNumber(index int) float64 {
	return float64(C.lua_tonumber(L.s, C.int(index)))
}

// lua_yield
/*
 * [-?, +?, -]
 * Yields a coroutine.
 */
func (L *State) Yield(nresults int) int {
	return int(C.lua_yield(L.s, C.int(nresults)))
}

func (L *State) pcall(nargs, nresults, errfunc int) int {
	return int(C.lua_pcall(L.s, C.int(nargs), C.int(nresults), C.int(errfunc)))
}

// Pushes on the stack the value of a global variable (lua_getglobal)
func (L *State) GetGlobal(name string) { L.GetField(LUA_GLOBALSINDEX, name) }

// lua_resume
/*
 * [-?, +?, -]
 * Starts and resumes a coroutine in a given thread.
 */
func (L *State) Resume(narg int) int {
	return int(C.lua_resume(L.s, C.int(narg)))
}

// lua_setglobal
/*
 * [-1, +0, e]
 * Pops a value from the stack and sets it as the new value of global name. It is defined as a macro:
 */
func (L *State) SetGlobal(name string) {
	Cname := C.CString(name)
	defer C.free(unsafe.Pointer(Cname))
	C.lua_setfield(L.s, C.int(LUA_GLOBALSINDEX), Cname)
}

// lua_insert
/*
 * [-1, +1, -]
 * Moves the top element into the given valid index, shifting up the elements above this index to open space. Cannot be called with a pseudo-index, because a pseudo-index is not an actual stack position.
 */
func (L *State) Insert(index int) { C.lua_insert(L.s, C.int(index)) }

// lua_remove
/*
 * [-1, +0, -]
 * Removes the element at the given valid index, shifting down the elements above this index to fill the gap. Cannot be called with a pseudo-index, because a pseudo-index is not an actual stack position.
 */
func (L *State) Remove(index int) {
	C.lua_remove(L.s, C.int(index))
}

// lua_replace
/*
 * [-1, +0, -]
 * Moves the top element into the given position (and pops it), without shifting any element (therefore replacing the value at the given position).
 */
func (L *State) Replace(index int) {
	C.lua_replace(L.s, C.int(index))
}

// lua_rawgeti
/*
 * [-0, +1, -]
 * Pushes onto the stack the value t[n], where t is the value at the given valid index. The access is raw; that is, it does not invoke metamethods.
 */
func (L *State) RawGeti(index int, n int) {
	C.lua_rawgeti(L.s, C.int(index), C.int(n))
}

// lua_rawseti
/*
 * [-1, +0, m]
 * Does the equivalent of t[n] = v, where t is the value at the given valid index and v is the value at the top of the stack.
 */
func (L *State) RawSeti(index int, n int) {
	C.lua_rawseti(L.s, C.int(index), C.int(n))
}

// lua_gc
/*
 * [-0, +0, e]
 * Controls the garbage collector.
 */
func (L *State) GC(what, data int) int {
	return int(C.lua_gc(L.s, C.int(what), C.int(data)))
}
