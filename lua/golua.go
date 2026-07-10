package lua

/*
#cgo !lua52,!lua53,!lua54 CFLAGS: -I ${SRCDIR}/lua51
#cgo lua52 CFLAGS: -I ${SRCDIR}/lua52
#cgo lua53 CFLAGS: -I ${SRCDIR}/lua53
#cgo lua54 CFLAGS: -I ${SRCDIR}/lua54

#include <lua.h>
#include <lualib.h>
#include <stdlib.h>
*/
import "C"

import (
	"context"
	"reflect"
	"sync"
	"unsafe"
)

// Type of allocation functions to use with NewStateAlloc
type Alloc func(ptr unsafe.Pointer, osize uint, nsize uint) unsafe.Pointer

// This is the type of go function that can be registered as lua functions
type LuaGoFunction func(L *State) int

// This is the type of a go function that can be used as a lua_Hook
type HookFunction func(L *State)

// The errorstring used by State.SetExecutionLimit
const ExecutionQuantumExceeded = "Lua execution quantum exceeded"

// Wrapper to keep cgo from complaining about incomplete ptr type
//
//export State
type State struct {
	// Wrapped lua_State object
	s *C.lua_State

	// index of this object inside the goStates array
	Index uintptr

	// Registry of go object that have been pushed to Lua VM
	registry []interface{}

	// Freelist for funcs indices, to allow for freeing
	freeIndices []uint

	// User self defined memory alloc func for the lua State
	allocfn *Alloc

	// User defined hook function
	hookFn HookFunction

	ctx context.Context

	// main points to the State that owns the Go-side bookkeeping (function
	// registry, free-list, hook function). It is nil for the main state
	// itself and non-nil for coroutine thread wrappers (created by NewThread
	// or synthesized by the C->Go dispatch when a Go function or hook is
	// invoked on a coroutine's lua_State). Thread wrappers share the parent's
	// Index and are never registered in goStates; all registry mutations are
	// delegated to main so that function ids stay valid across threads.
	main *State
}

// mainState returns the State that owns the Go-side bookkeeping: the receiver
// itself for a main state, or the creating state for a coroutine thread
// wrapper. All registry/free-list/hook accesses must go through it.
func (L *State) mainState() *State {
	if L.main != nil {
		return L.main
	}
	return L
}

// getGoStateForThread resolves the *State wrapper for the actual calling
// lua_State. C->Go dispatch identifies the VM by the registry-stored index of
// the main state (shared by all coroutine threads of that VM); when the
// calling lua_State is a coroutine, a transient wrapper is synthesized so the
// Go callback reads arguments from and pushes results onto the CORRECT stack.
// The wrapper is transient (one per dispatch): no caching, hence no
// invalidation hazard when threads are collected; the allocation is noise
// next to the cgo crossing itself.
func getGoStateForThread(gostateindex uintptr, ls *C.lua_State) *State {
	M := getGoState(gostateindex)
	if M == nil || M.s == ls {
		return M
	}
	return &State{s: ls, Index: M.Index, main: M}
}

var (
	goStates      map[uintptr]*State
	goStatesMutex sync.Mutex
)

func init() {
	goStates = make(map[uintptr]*State, 16)
}

func registerGoState(L *State) {
	goStatesMutex.Lock()
	defer goStatesMutex.Unlock()
	L.Index = uintptr(unsafe.Pointer(L))
	goStates[L.Index] = L
}

func unregisterGoState(L *State) {
	goStatesMutex.Lock()
	defer goStatesMutex.Unlock()
	delete(goStates, L.Index)
}

func getGoState(gostateindex uintptr) *State {
	goStatesMutex.Lock()
	defer goStatesMutex.Unlock()
	return goStates[gostateindex]
}

//export golua_callgofunction
func golua_callgofunction(gostateindex uintptr, ls *C.lua_State, fid uint) int {
	// ls is the lua_State the call actually happens on: the main state or a
	// coroutine thread. The Go function must see THAT stack (its arguments
	// live there), while the function registry is looked up on the main state.
	L1 := getGoStateForThread(gostateindex, ls)
	if fid < 0 {
		panic(&LuaError{0, "Requested execution of an unknown function", L1.StackTrace()})
	}
	f := L1.mainState().registry[fid].(LuaGoFunction)
	return f(L1)
}

//export golua_callgohook
func golua_callgohook(gostateindex uintptr, ls *C.lua_State) {
	// Deliver the hook on the thread it fired on (correct stack for
	// inspection); the hook function itself is shared VM bookkeeping.
	L1 := getGoStateForThread(gostateindex, ls)
	if L1 == nil {
		return
	}
	if hf := L1.mainState().hookFn; hf != nil {
		hf(L1)
	}
}

var typeOfBytes = reflect.TypeOf([]byte(nil))

//export golua_interface_newindex_callback
func golua_interface_newindex_callback(gostateindex uintptr, ls *C.lua_State, iid uint, field_name_cstr *C.char) int {
	L := getGoStateForThread(gostateindex, ls)
	iface := L.mainState().registry[iid]
	ifacevalue := reflect.ValueOf(iface).Elem()

	field_name := C.GoString(field_name_cstr)

	fval := ifacevalue.FieldByName(field_name)

	if fval.Kind() == reflect.Ptr {
		fval = fval.Elem()
	}

	luatype := LuaValType(C.lua_type(L.s, 3))

	switch fval.Kind() {
	case reflect.Bool:
		if luatype == LUA_TBOOLEAN {
			fval.SetBool(int(C.lua_toboolean(L.s, 3)) != 0)
			return 1
		} else {
			L.PushString("Wrong assignment to field " + field_name)
			return -1
		}

	case reflect.Int:
		fallthrough
	case reflect.Int8:
		fallthrough
	case reflect.Int16:
		fallthrough
	case reflect.Int32:
		fallthrough
	case reflect.Int64:
		if luatype == LUA_TNUMBER {
			fval.SetInt(int64(luaToInteger(L.s, 3)))
			return 1
		} else {
			L.PushString("Wrong assignment to field " + field_name)
			return -1
		}

	case reflect.Uint:
		fallthrough
	case reflect.Uint8:
		fallthrough
	case reflect.Uint16:
		fallthrough
	case reflect.Uint32:
		fallthrough
	case reflect.Uint64:
		if luatype == LUA_TNUMBER {
			fval.SetUint(uint64(luaToInteger(L.s, 3)))
			return 1
		} else {
			L.PushString("Wrong assignment to field " + field_name)
			return -1
		}

	case reflect.String:
		if luatype == LUA_TSTRING {
			fval.SetString(C.GoString(C.lua_tolstring(L.s, 3, nil)))
			return 1
		} else {
			L.PushString("Wrong assignment to field " + field_name)
			return -1
		}

	case reflect.Float32:
		fallthrough
	case reflect.Float64:
		if luatype == LUA_TNUMBER {
			fval.SetFloat(float64(luaToNumber(L.s, 3)))
			return 1
		} else {
			L.PushString("Wrong assignment to field " + field_name)
			return -1
		}
	case reflect.Slice:
		if fval.Type() == typeOfBytes {
			if luatype == LUA_TSTRING {
				fval.SetBytes(L.ToBytes(3))
				return 1
			} else {
				L.PushString("Wrong assignment to field " + field_name)
				return -1
			}
		}
	}

	L.PushString("Unsupported type of field " + field_name + ": " + fval.Type().String())
	return -1
}

//export golua_interface_index_callback
func golua_interface_index_callback(gostateindex uintptr, ls *C.lua_State, iid uint, field_name *C.char) int {
	L := getGoStateForThread(gostateindex, ls)
	iface := L.mainState().registry[iid]
	ifacevalue := reflect.ValueOf(iface).Elem()

	fval := ifacevalue.FieldByName(C.GoString(field_name))

	if fval.Kind() == reflect.Ptr {
		fval = fval.Elem()
	}

	switch fval.Kind() {
	case reflect.Bool:
		L.PushBoolean(fval.Bool())
		return 1

	case reflect.Int:
		fallthrough
	case reflect.Int8:
		fallthrough
	case reflect.Int16:
		fallthrough
	case reflect.Int32:
		fallthrough
	case reflect.Int64:
		L.PushInteger(fval.Int())
		return 1

	case reflect.Uint:
		fallthrough
	case reflect.Uint8:
		fallthrough
	case reflect.Uint16:
		fallthrough
	case reflect.Uint32:
		fallthrough
	case reflect.Uint64:
		L.PushInteger(int64(fval.Uint()))
		return 1

	case reflect.String:
		L.PushString(fval.String())
		return 1

	case reflect.Float32:
		fallthrough
	case reflect.Float64:
		L.PushNumber(fval.Float())
		return 1
	case reflect.Slice:
		if fval.Type() == typeOfBytes {
			L.PushBytes(fval.Bytes())
			return 1
		}
	}

	L.PushString("Unsupported type of field: " + fval.Type().String())
	return -1
}

//export golua_gchook
func golua_gchook(gostateindex uintptr, id uint) int {
	L1 := getGoState(gostateindex)
	L1.unregister(id)
	return 0
}

//export golua_callpanicfunction
func golua_callpanicfunction(gostateindex uintptr, ls *C.lua_State, id uint) int {
	L1 := getGoStateForThread(gostateindex, ls)
	f := L1.mainState().registry[id].(LuaGoFunction)
	return f(L1)
}

//export golua_idtointerface
func golua_idtointerface(id uint) interface{} {
	return id
}

//export golua_cfunctiontointerface
func golua_cfunctiontointerface(f *uintptr) interface{} {
	return f
}

//export golua_callallocf
func golua_callallocf(fp uintptr, ptr uintptr, osize uint, nsize uint) uintptr {
	return uintptr((*((*Alloc)(unsafe.Pointer(fp))))(unsafe.Pointer(ptr), osize, nsize))
}

//export go_panic_msghandler
func go_panic_msghandler(gostateindex uintptr, ls *C.lua_State, z *C.char) {
	L := getGoStateForThread(gostateindex, ls)
	s := C.GoString(z)

	panic(&LuaError{LUA_ERRERR, s, L.StackTrace()})
}
