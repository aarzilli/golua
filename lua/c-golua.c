#include <lua.h>
#include <lauxlib.h>
#include <lualib.h>
#include <stdint.h>
#include "_cgo_export.h"
//metatables to register:
//	GoLua.GoInterface
//  GoLua.GoFunction
//

static const char GoStateRegistryKey = 'k'; //golua registry key
static const char PanicFIDRegistryKey = 'k';


unsigned int* clua_checkgofunction(lua_State* L, int index)
{
	unsigned int* fid = (unsigned int*)luaL_checkudata(L,index,"GoLua.GoFunction");
	luaL_argcheck(L, fid != NULL, index, "'GoFunction' expected");
	return fid;
}

GoInterface* clua_getgostate(lua_State* L)
{
	//get gostate from registry entry
	lua_pushlightuserdata(L,(void*)&GoStateRegistryKey);
	lua_gettable(L, LUA_REGISTRYINDEX);
	GoInterface* gip = lua_touserdata(L,-1);
	lua_pop(L,1);
	return gip;
}


//wrapper for callgofunction
int callback_function(lua_State* L)
{
	unsigned int *fid = clua_checkgofunction(L,1);
	GoInterface* gi = clua_getgostate(L);
	//remove the go function from the stack (to present same behavior as lua_CFunctions)
	lua_remove(L,1);
	return golua_callgofunction(*gi,*fid);
}

//wrapper for gchook
int gchook_wrapper(lua_State* L)
{
	unsigned int* fid = clua_checkgofunction(L,-1); //TODO: this will error
	GoInterface* gi = clua_getgostate(L);
	if(fid != NULL)
		return golua_gchook(*gi,*fid);
	printf("GCHook failed\n");
	//TODO: try udata or whatever, after impl

	return 0;
}

unsigned int clua_togofunction(lua_State* L, int index)
{
	return *(clua_checkgofunction(L,index));
}

void clua_pushgofunction(lua_State* L, unsigned int fid)
{
	unsigned int* fidptr = (unsigned int*)lua_newuserdata(L, sizeof(unsigned int));
	*fidptr = fid;
	luaL_getmetatable(L, "GoLua.GoFunction");
	lua_setmetatable(L, -2);
}

void clua_pushlightinteger(lua_State* L, int n)
{
  lua_pushlightuserdata(L, (void*)(GoUintptr)n);
}

GoUintptr clua_tolightinteger(lua_State *L, unsigned int index)
{
  return (GoUintptr)lua_touserdata(L, index);
}

void clua_setgostate(lua_State* L, GoInterface gi)
{
	lua_pushlightuserdata(L,(void*)&GoStateRegistryKey);
	GoInterface* gip = (GoInterface*)lua_newuserdata(L,sizeof(GoInterface));
	//copy interface value to userdata
	gip->v = gi.v;
	gip->t = gi.t;
	//set into registry table
	lua_settable(L,LUA_REGISTRYINDEX);

}


void clua_pushgointerface(lua_State* L, GoInterface gi)
{
	GoInterface* iptr = (GoInterface*)lua_newuserdata(L, sizeof(GoInterface));
	iptr->v = gi.v;
	iptr->t = gi.t;
	luaL_getmetatable(L, "GoLua.GoInterface");
	lua_setmetatable(L,-2);
}

void clua_initstate(lua_State* L)
{
	/* create the GoLua.GoFunction metatable */
	luaL_newmetatable(L,"GoLua.GoFunction");
	//pushkey
	lua_pushliteral(L,"__call");
	//push value
	lua_pushcfunction(L,&callback_function);
	//t[__call] = &callback_function
	lua_settable(L,-3);
	//push key
	lua_pushliteral(L,"__gc");
	//pushvalue
	lua_pushcfunction(L,&gchook_wrapper);
	lua_settable(L,-3);
	lua_pop(L,1);
}


int callback_panicf(lua_State* L)
{
	lua_pushlightuserdata(L,(void*)&PanicFIDRegistryKey);
	lua_gettable(L,LUA_REGISTRYINDEX);
	unsigned int fid = lua_tointeger(L,-1);
	lua_pop(L,1);
	GoInterface* gi = clua_getgostate(L);
	return golua_callpanicfunction(*gi,fid);

}

//TODO: currently setting garbage when panicf set to null
GoInterface clua_atpanic(lua_State* L, unsigned int panicf_id)
{
	//get old panicfid
	unsigned int old_id;
	lua_pushlightuserdata(L, (void*)&PanicFIDRegistryKey);
	lua_gettable(L,LUA_REGISTRYINDEX);
	if(lua_isnil(L,-1) == 0)
		old_id = lua_tointeger(L,-1);
	lua_pop(L,1);

	//set registry key for function id of go panic function
	lua_pushlightuserdata(L,(void*)&PanicFIDRegistryKey);
	//push id value
	lua_pushinteger(L,panicf_id);
	//set into registry table
	lua_settable(L,LUA_REGISTRYINDEX);

	//now set the panic function
	lua_CFunction pf = lua_atpanic(L,&callback_panicf);
	//make a GoInterface with a wrapped C panicf or the original go panicf
	if(pf == &callback_panicf)
	{
		return golua_idtointerface(old_id);
	}
	else
	{
		//TODO: technically UB, function ptr -> non function ptr
		return golua_cfunctiontointerface((GoUintptr *)pf);
	}
}

int clua_callluacfunc(lua_State* L, lua_CFunction f)
{
	return f(L);
}

void* allocwrapper(void* ud, void *ptr, size_t osize, size_t nsize)
{
	return (void*)golua_callallocf((GoUintptr)ud,(GoUintptr)ptr,osize,nsize);
}

lua_State* clua_newstate(void* goallocf)
{
	return lua_newstate(&allocwrapper,goallocf);
}

void clua_setallocf(lua_State* L, void* goallocf)
{
	lua_setallocf(L,&allocwrapper,goallocf);
}

void clua_openbase(lua_State* L){
	lua_pushcfunction(L,&luaopen_base);
	lua_pushstring(L,"");
	lua_call(L, 1, 0);
}

void clua_openio(lua_State* L){
	lua_pushcfunction(L,&luaopen_io);
	lua_pushstring(L,"io");
	lua_call(L, 1, 0);
}

void clua_openmath(lua_State* L){
	lua_pushcfunction(L,&luaopen_math);
	lua_pushstring(L,"math");
	lua_call(L, 1, 0);
}

void clua_openpackage(lua_State* L){
	lua_pushcfunction(L,&luaopen_package);
	lua_pushstring(L,"package");
	lua_call(L, 1, 0);
}

void clua_openstring(lua_State* L){
	lua_pushcfunction(L,&luaopen_string);
	lua_pushstring(L,"string");
	lua_call(L, 1, 0);
}

void clua_opentable(lua_State* L){
	lua_pushcfunction(L,&luaopen_table);
	lua_pushstring(L,"table");
	lua_call(L, 1, 0);
}

void clua_openos(lua_State* L){
	lua_pushcfunction(L,&luaopen_os);
	lua_pushstring(L,"os");
	lua_call(L, 1, 0);
}

void clua_hook_function(lua_State *L, lua_Debug *ar) {
  lua_checkstack(L, 2);
  lua_pushstring(L, "Lua execution quantum exceeded");
  lua_error(L);
}

void clua_setexecutionlimit(lua_State* L, int n) {
  lua_sethook(L, &clua_hook_function, LUA_MASKCOUNT, n);
}

