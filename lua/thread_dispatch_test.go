package lua

import (
	"testing"
)

// The tests in this file cover thread-correct dispatch of Go callbacks: a Go
// function (or hook) invoked from a coroutine must receive a *State bound to
// the coroutine's lua_State (its arguments live on that stack), while the
// function registry stays on the main state.

// TestGoFunctionFromCoroutineUsesCallingThreadStack verifies that a Go function called from inside
// coroutine.wrap reads its arguments from the coroutine stack and returns
// results onto it. Before the thread-dispatch fix, the function received the
// main state's wrapper: ToInteger(1) read the wrong (empty) stack and the
// result was pushed onto the wrong stack.
func TestGoFunctionFromCoroutineUsesCallingThreadStack(t *testing.T) {
	L := NewState()
	defer L.Close()
	L.OpenLibs()

	var sawThreadState bool
	var gotTop int
	var gotArg int64

	L.Register("double", func(l *State) int {
		sawThreadState = l.s != L.s // must run on the coroutine's lua_State
		gotTop = l.GetTop()
		gotArg = int64(l.ToInteger(1))
		l.PushInteger(gotArg * 2)
		return 1
	})

	err := L.DoString(`
		local co = coroutine.wrap(function(a) return double(a + 1) end)
		result = co(20)
	`)
	if err != nil {
		t.Fatalf("DoString failed: %v", err)
	}

	L.GetGlobal("result")
	if got := L.ToInteger(-1); got != 42 {
		t.Errorf("expected result 42, got %d", got)
	}
	L.Pop(1)

	if !sawThreadState {
		t.Error("Go function ran with the main state's wrapper, not the coroutine's")
	}
	if gotTop != 1 {
		t.Errorf("expected GetTop()==1 inside Go function, got %d", gotTop)
	}
	if gotArg != 21 {
		t.Errorf("expected argument 21 inside Go function, got %d", gotArg)
	}
}

// TestGoFunctionFromCoroutineMultiArgs verifies multi-argument reads and
// multi-value returns across the coroutine boundary.
func TestGoFunctionFromCoroutineMultiArgs(t *testing.T) {
	L := NewState()
	defer L.Close()
	L.OpenLibs()

	L.Register("swap", func(l *State) int {
		a := l.ToString(1)
		b := l.ToString(2)
		l.PushString(b)
		l.PushString(a)
		return 2
	})

	err := L.DoString(`
		local co = coroutine.wrap(function() return swap("first", "second") end)
		x, y = co()
	`)
	if err != nil {
		t.Fatalf("DoString failed: %v", err)
	}

	L.GetGlobal("x")
	L.GetGlobal("y")
	if x := L.ToString(-2); x != "second" {
		t.Errorf("expected x=='second', got %q", x)
	}
	if y := L.ToString(-1); y != "first" {
		t.Errorf("expected y=='first', got %q", y)
	}
	L.Pop(2)
}

// TestPushGoFunctionInsideCoroutineSharesMainRegistry verifies that registering a new Go
// function from a Go function running on a coroutine delegates to the main
// state's registry (a thread wrapper has no registry of its own).
func TestPushGoFunctionInsideCoroutineSharesMainRegistry(t *testing.T) {
	L := NewState()
	defer L.Close()
	L.OpenLibs()

	L.Register("makeadder", func(l *State) int {
		n := int64(l.ToInteger(1))
		l.PushGoFunction(func(l2 *State) int {
			l2.PushInteger(int64(l2.ToInteger(1)) + n)
			return 1
		})
		return 1
	})

	err := L.DoString(`
		local co = coroutine.wrap(function()
			local add5 = makeadder(5)
			return add5(37)
		end)
		result = co()
	`)
	if err != nil {
		t.Fatalf("DoString failed: %v", err)
	}

	L.GetGlobal("result")
	if got := L.ToInteger(-1); got != 42 {
		t.Errorf("expected result 42, got %d", got)
	}
	L.Pop(1)
}

// TestNewThreadDispatchesGoFunctionOnItsOwnStack verifies the C-API path: a thread created
// via NewThread runs a Lua function that calls a Go function; the Go function
// must operate on the thread's stack, and results must be readable from the
// thread wrapper after Resume.
func TestNewThreadDispatchesGoFunctionOnItsOwnStack(t *testing.T) {
	L := NewState()
	defer L.Close()
	L.OpenLibs()

	L.Register("triple", func(l *State) int {
		if l.s == L.s {
			t.Error("Go function ran on the main state instead of the thread")
		}
		l.PushInteger(int64(l.ToInteger(1)) * 3)
		return 1
	})

	co := L.NewThread() // thread object is now on L's stack; keep it there (GC anchor)

	if rc := co.LoadString(`return triple(14)`); rc != 0 {
		t.Fatalf("LoadString on thread failed: rc=%d", rc)
	}
	if rc := co.Resume(0); rc != 0 {
		t.Fatalf("Resume failed: rc=%d (%s)", rc, co.ToString(-1))
	}
	if got := co.ToInteger(-1); got != 42 {
		t.Errorf("expected 42 from thread, got %d", got)
	}
	co.Pop(1)
	L.Pop(1) // pop the thread object from the main stack
}

// TestHookReceivesCallingThreadState verifies that a count hook firing while a coroutine
// runs receives a wrapper bound to the coroutine's lua_State.
func TestHookReceivesCallingThreadState(t *testing.T) {
	L := NewState()
	defer L.Close()
	L.OpenLibs()

	var hookFired, hookSawThread bool
	L.SetHook(func(l *State) {
		hookFired = true
		if l.s != L.s {
			hookSawThread = true
		}
	}, 10)
	defer L.SetHook(nil, 0)

	err := L.DoString(`
		local co = coroutine.wrap(function()
			local s = 0
			for i = 1, 10000 do s = s + i end
			return s
		end)
		co()
	`)
	if err != nil {
		t.Fatalf("DoString failed: %v", err)
	}

	if !hookFired {
		t.Skip("count hook never fired (JIT-compiled trace?); dispatch not exercised")
	}
	if !hookSawThread {
		t.Error("hook never observed the coroutine's lua_State")
	}
}

// TestGoClosureFromCoroutineUsesCallingThreadStack covers the callback_c
// dispatch path (clua_pushcallback, used by PushGoClosure): a Go closure
// called from a coroutine must read its arguments from the coroutine's stack,
// exactly like plain Go functions on the callback_function path.
func TestGoClosureFromCoroutineUsesCallingThreadStack(t *testing.T) {
	L := NewState()
	defer L.Close()
	L.OpenLibs()

	var sawThreadState bool
	L.PushGoClosure(func(l *State) int {
		sawThreadState = l.s != L.s
		l.PushInteger(int64(l.ToInteger(1)) + 1)
		return 1
	})
	L.SetGlobal("incr")

	err := L.DoString(`
		local co = coroutine.wrap(function(a) return incr(a) end)
		closure_result = co(41)
	`)
	if err != nil {
		t.Fatalf("DoString failed: %v", err)
	}
	L.GetGlobal("closure_result")
	if got := L.ToInteger(-1); got != 42 {
		t.Errorf("expected 42, got %d", got)
	}
	L.Pop(1)
	if !sawThreadState {
		t.Error("Go closure ran with the main state's wrapper, not the coroutine's")
	}
}

// TestGoStructAccessFromCoroutineUsesCallingThreadStack covers the
// interface_index/interface_newindex dispatch paths: reading and assigning a
// GoStruct field from inside a coroutine must operate on the coroutine's
// stack (the assigned value lives there; the read result must be pushed
// there).
func TestGoStructAccessFromCoroutineUsesCallingThreadStack(t *testing.T) {
	type point struct {
		N int
	}

	L := NewState()
	defer L.Close()
	L.OpenLibs()

	p := &point{N: 7}
	L.PushGoStruct(p)
	L.SetGlobal("pt")

	err := L.DoString(`
		local co = coroutine.wrap(function()
			local before = pt.N   -- __index from the coroutine
			pt.N = before + 35    -- __newindex reads the value from the coroutine's stack
			return before
		end)
		struct_before = co()
	`)
	if err != nil {
		t.Fatalf("DoString failed: %v", err)
	}
	L.GetGlobal("struct_before")
	if got := L.ToInteger(-1); got != 7 {
		t.Errorf("expected to read 7 inside the coroutine, got %d", got)
	}
	L.Pop(1)
	if p.N != 42 {
		t.Errorf("expected the assignment from the coroutine to land (42), got %d", p.N)
	}
}
