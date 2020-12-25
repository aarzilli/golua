//+build !lua52,!lua53,!lua54

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
func (L *State) Equal(index1, index2 int) bool {
	return C.lua_equal(L.s, C.int(index1), C.int(index2)) == 1
}

// lua_getfenv
func (L *State) GetfEnv(index int) {
	C.lua_getfenv(L.s, C.int(index))
}

// lua_lessthan
func (L *State) LessThan(index1, index2 int) bool {
	return C.lua_lessthan(L.s, C.int(index1), C.int(index2)) == 1
}

// lua_setfenv
func (L *State) SetfEnv(index int) {
	C.lua_setfenv(L.s, C.int(index))
}

func (L *State) ObjLen(index int) uint {
	return uint(C.lua_objlen(L.s, C.int(index)))
}

// lua_tointeger
func (L *State) ToInteger(index int) int {
	return int(C.lua_tointeger(L.s, C.int(index)))
}

// lua_tonumber
func (L *State) ToNumber(index int) float64 {
	return float64(C.lua_tonumber(L.s, C.int(index)))
}

// lua_yield
func (L *State) Yield(nresults int) int {
	return int(C.lua_yield(L.s, C.int(nresults)))
}

func (L *State) pcall(nargs, nresults, errfunc int) int {
	return int(C.lua_pcall(L.s, C.int(nargs), C.int(nresults), C.int(errfunc)))
}

// Pushes on the stack the value of a global variable (lua_getglobal)
func (L *State) GetGlobal(name string) { L.GetField(LUA_GLOBALSINDEX, name) }

// lua_resume
func (L *State) Resume(narg int) int {
	return int(C.lua_resume(L.s, C.int(narg)))
}

// lua_setglobal
func (L *State) SetGlobal(name string) {
	Cname := C.CString(name)
	defer C.free(unsafe.Pointer(Cname))
	C.lua_setfield(L.s, C.int(LUA_GLOBALSINDEX), Cname)
}

// lua_insert
func (L *State) Insert(index int) { C.lua_insert(L.s, C.int(index)) }

// lua_remove
func (L *State) Remove(index int) {
	C.lua_remove(L.s, C.int(index))
}

// lua_replace
func (L *State) Replace(index int) {
	C.lua_replace(L.s, C.int(index))
}

// lua_rawgeti
func (L *State) RawGeti(index int, n int) {
	C.lua_rawgeti(L.s, C.int(index), C.int(n))
}

// lua_rawseti
func (L *State) RawSeti(index int, n int) {
	C.lua_rawseti(L.s, C.int(index), C.int(n))
}

// lua_gc
func (L *State) GC(what, data int) int {
	return int(C.lua_gc(L.s, C.int(what), C.int(data)))
}
