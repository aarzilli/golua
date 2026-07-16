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
	"fmt"
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
	// Wrapped lua_State object: the per-thread execution stack (the main
	// state or one coroutine thread).
	s *C.lua_State

	// key of the owning sharedState inside the goStates map (equal to the
	// address of that sharedState). All threads of a VM share this value.
	Index uintptr

	// shared points to the VM-wide Go-side bookkeeping shared by the main
	// state and every coroutine thread wrapper of that VM. It is the same
	// instance for all of them, so there is no per-thread copy to touch by
	// mistake: every registry/free-list/hook/ctx access goes through here.
	shared *sharedState
}

// sharedState holds the Go-side bookkeeping owned by a Lua VM and shared by
// its main state and every coroutine thread wrapper. Exactly one instance per
// VM lives in the goStates map; State is a thin per-thread wrapper pointing at
// it. Keeping it a distinct type means a stray direct field access on a thread
// wrapper is impossible (the fields simply do not exist on State) rather than a
// silent read of an empty per-thread slice.
type sharedState struct {
	// Registry of go object that have been pushed to Lua VM
	registry []interface{}

	// Freelist for funcs indices, to allow for freeing
	freeIndices []uint

	// User self defined memory alloc func for the lua State
	allocfn *Alloc

	// User defined hook function
	hookFn HookFunction

	ctx context.Context

	// pendingTrace holds the Lua stack trace captured while the erroring
	// stack was still alive (by the message handler, or at the point a Go
	// panic was converted to a lua_error). callEx consumes it to build the
	// LuaError after lua_pcall has already unwound the stack.
	pendingTrace []LuaStackEntry

	// pendingPanic preserves the ORIGINAL value recovered from a panicking
	// Go callback (RaiseError's *LuaError, a user's custom error, ...) while
	// the error itself travels through the VM as a regular lua_error.
	// callEx returns this original value — matched via pendingPanicMsg
	// against the error string on the stack — so existing Go code observes
	// exactly the pre-fix results from Call/DoString. If a Lua-level pcall
	// catches the error instead, the stale value is discarded by the
	// message-mismatch guard on the next error.
	pendingPanic    interface{}
	pendingPanicMsg string
}

// getGoStateForThread resolves a *State wrapper for the actual calling
// lua_State. C->Go dispatch identifies the VM by the goStates key stored in the
// registry (shared by all coroutine threads of that VM); a fresh wrapper is
// synthesized around the CALLING lua_State so the Go callback reads arguments
// from and pushes results onto the correct stack, while all bookkeeping is
// reached through the shared sharedState. The wrapper is transient (one per
// dispatch, main state included): no caching, hence no invalidation hazard when
// threads are collected; the allocation is noise next to the cgo crossing.
func getGoStateForThread(gostateindex uintptr, ls *C.lua_State) *State {
	sh := getGoState(gostateindex)
	if sh == nil {
		return nil
	}
	return &State{s: ls, Index: gostateindex, shared: sh}
}

var (
	goStates      map[uintptr]*sharedState
	goStatesMutex sync.Mutex
)

func init() {
	goStates = make(map[uintptr]*sharedState, 16)
}

// registerGoState stores the VM's sharedState in the goStates map, keyed by its
// own address, and returns that key for the main state's Index. The key
// round-trips through C as an opaque size_t (never converted back to a Go
// pointer), so it is used solely as the map lookup key on the way back.
func registerGoState(sh *sharedState) uintptr {
	goStatesMutex.Lock()
	defer goStatesMutex.Unlock()
	idx := uintptr(unsafe.Pointer(sh))
	goStates[idx] = sh
	return idx
}

func unregisterGoState(idx uintptr) {
	goStatesMutex.Lock()
	defer goStatesMutex.Unlock()
	delete(goStates, idx)
}

func getGoState(gostateindex uintptr) *sharedState {
	goStatesMutex.Lock()
	defer goStatesMutex.Unlock()
	return goStates[gostateindex]
}

// pushRecoveredError converts a value recovered from a panicking Go callback
// into a Lua error value on top of L's stack and returns the sentinel -1 that
// tells the C trampoline to raise it with lua_error. Raising from C keeps the
// longjmp entirely inside the Lua VM: a Go panic must never unwind across the
// in-flight lua_pcall/lua_resume C frames, as that abandons the VM's internal
// call chain and corrupts the state (the error is then only observable as a
// later crash). The stack trace is preserved via pendingTrace for callEx.
func (L *State) pushRecoveredError(r interface{}) int {
	sh := L.shared
	L.CheckStack(1)
	var msg string
	switch e := r.(type) {
	case *LuaError:
		sh.pendingTrace = e.stackTrace
		msg = e.message
	case error:
		sh.pendingTrace = L.StackTrace()
		msg = e.Error()
	default:
		sh.pendingTrace = L.StackTrace()
		msg = fmt.Sprintf("panic: %v", r)
	}
	sh.pendingPanic = r
	sh.pendingPanicMsg = msg
	L.PushString(msg)
	return -1
}

//export golua_callgofunction
func golua_callgofunction(gostateindex uintptr, ls *C.lua_State, fid uint) (nret int) {
	// ls is the lua_State the call actually happens on: the main state or a
	// coroutine thread. The Go function must see THAT stack (its arguments
	// live there), while the function registry is looked up on the main state.
	L1 := getGoStateForThread(gostateindex, ls)
	// Contain any panic (RaiseError or accidental) at the cgo boundary and
	// let the C trampoline raise it as a regular lua_error instead of
	// unwinding through the VM's C frames (see pushRecoveredError).
	defer func() {
		if r := recover(); r != nil {
			nret = L1.pushRecoveredError(r)
		}
	}()
	if fid < 0 {
		panic(&LuaError{0, "Requested execution of an unknown function", L1.StackTrace()})
	}
	f := L1.shared.registry[fid].(LuaGoFunction)
	return f(L1)
}

//export golua_callgohook
func golua_callgohook(gostateindex uintptr, ls *C.lua_State) (raise int) {
	// Deliver the hook on the thread it fired on (correct stack for
	// inspection); the hook function itself is shared VM bookkeeping.
	L1 := getGoStateForThread(gostateindex, ls)
	if L1 == nil {
		return 0
	}
	// A panicking hook (e.g. SetExecutionLimit's RaiseError) must not unwind
	// across the VM's C frames either: convert it to a lua_error raised by
	// the C hook wrapper, which resolves inside the interrupted pcall/resume.
	defer func() {
		if r := recover(); r != nil {
			L1.pushRecoveredError(r)
			raise = 1
		}
	}()
	if hf := L1.shared.hookFn; hf != nil {
		hf(L1)
	}
	return 0
}

var typeOfBytes = reflect.TypeOf([]byte(nil))

//export golua_interface_newindex_callback
func golua_interface_newindex_callback(gostateindex uintptr, ls *C.lua_State, iid uint, field_name_cstr *C.char) (r int) {
	L := getGoStateForThread(gostateindex, ls)
	// Contain reflect panics (Elem on a non-pointer, Set on a wrong kind, ...)
	// like any other Go panic: the C wrapper raises the pushed value on r < 0.
	defer func() {
		if rec := recover(); rec != nil {
			r = L.pushRecoveredError(rec)
		}
	}()
	iface := L.shared.registry[iid]
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
func golua_interface_index_callback(gostateindex uintptr, ls *C.lua_State, iid uint, field_name *C.char) (r int) {
	L := getGoStateForThread(gostateindex, ls)
	// Contain reflect panics (Elem on a non-pointer, ...) like any other Go
	// panic: the C wrapper raises the pushed value on r < 0.
	defer func() {
		if rec := recover(); rec != nil {
			r = L.pushRecoveredError(rec)
		}
	}()
	iface := L.shared.registry[iid]
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
	sh := getGoState(gostateindex)
	sh.unregister(id)
	return 0
}

//export golua_callpanicfunction
func golua_callpanicfunction(gostateindex uintptr, ls *C.lua_State, id uint) int {
	L1 := getGoStateForThread(gostateindex, ls)
	f := L1.shared.registry[id].(LuaGoFunction)
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
func go_panic_msghandler(gostateindex uintptr, ls *C.lua_State) {
	// Runs as lua_pcall's message handler while the erroring stack is still
	// alive: capture the trace for callEx and return normally. Panicking here
	// (the old behavior) unwound a Go panic across the in-flight lua_pcall —
	// the corruption class this file now exists to prevent. The error value
	// itself stays on the stack as the handler's result.
	L := getGoStateForThread(gostateindex, ls)
	L.shared.pendingTrace = L.StackTrace()
}
