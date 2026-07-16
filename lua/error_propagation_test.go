package lua

import (
	"errors"
	"strings"
	"testing"
)

// The tests in this file cover panic containment at the cgo boundary: a Go
// panic (RaiseError or accidental) raised inside a Go callback must be
// converted to a regular lua_error by the C trampoline and never unwind
// across the VM's in-flight lua_pcall/lua_resume frames. Afterwards the VM
// must remain fully usable.

func mustBeAlive(t *testing.T, L *State) {
	t.Helper()
	if err := L.DoString(`alive = (alive or 0) + 1`); err != nil {
		t.Fatalf("VM unusable after error: %v", err)
	}
}

// TestGoPanicsSurfaceFromCall verifies that a panic in a Go callback —
// whatever its payload type — surfaces from Call/DoString as an error
// carrying the payload, never as silent success (stock upstream swallowed
// non-error payloads: err stayed nil), with the VM intact afterwards.
func TestGoPanicsSurfaceFromCall(t *testing.T) {
	cases := []struct {
		name    string
		payload interface{}
		want    string // substring the surfaced error must carry
	}{
		{"string payload", "boom-payload", "boom-payload"},
		{"non-error int payload (swallowed by stock upstream)", 42, "42"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			L := NewState()
			defer L.Close()
			L.OpenLibs()

			L.Register("boom", func(l *State) int {
				panic(tc.payload)
			})

			err := L.DoString(`boom()`)
			if err == nil {
				t.Fatal("expected error, got nil (panic swallowed)")
			}
			if !strings.Contains(err.Error(), tc.want) {
				t.Errorf("error should carry the panic payload, got: %v", err)
			}
			mustBeAlive(t, L)
		})
	}
}

// TestGoErrorsAreCatchableInLua covers the full source×catcher matrix for
// errors raised in Go callbacks: both a deliberate RaiseError and an
// accidental panic MUST be containable by both Lua-level pcall (exposed as
// unsafe_pcall) and coroutine.resume, with the VM intact afterwards.
// Previously the underlying Go panic bypassed both and could only be
// recovered at the outermost Call boundary (issue #128, defect 2).
func TestGoErrorsAreCatchableInLua(t *testing.T) {
	sources := []struct {
		name string
		fn   LuaGoFunction
		want string // substring the catcher must observe
	}{
		{"RaiseError", func(l *State) int { l.RaiseError("deliberate-error"); return 0 }, "deliberate-error"},
		{"panic", func(l *State) int { panic("accidental-panic") }, "accidental-panic"},
	}
	catchers := []struct {
		name   string
		script string // calls global fail(), stores results in globals ok, msg
	}{
		{"pcall", `ok, msg = unsafe_pcall(fail)`},
		{"xpcall", `ok, msg = unsafe_xpcall(fail, function(m) return "handled: " .. tostring(m) end)`},
		{"resume", `local co = coroutine.create(function() fail() end)
			ok, msg = coroutine.resume(co)`},
	}
	for _, src := range sources {
		for _, cat := range catchers {
			t.Run(src.name+"_via_"+cat.name, func(t *testing.T) {
				L := NewState()
				defer L.Close()
				L.OpenLibs()

				L.Register("fail", src.fn)

				if err := L.DoString(cat.script); err != nil {
					t.Fatalf("the error bypassed %s and aborted the script: %v", cat.name, err)
				}
				L.GetGlobal("ok")
				if L.ToBoolean(-1) {
					t.Errorf("%s should have returned false", cat.name)
				}
				L.Pop(1)
				L.GetGlobal("msg")
				if msg := L.ToString(-1); !strings.Contains(msg, src.want) {
					t.Errorf("%s should have observed the error message, got %q", cat.name, msg)
				}
				L.Pop(1)
				mustBeAlive(t, L)
			})
		}
	}
}

// TestReflectPanicInInterfaceCallbackIsContained closes the containment gap
// found by the fix-completeness audit: the __index/__newindex callbacks for
// GoStruct values run reflect operations that can panic (e.g. Elem() when a
// non-pointer struct was pushed). Such a panic must surface as a regular Lua
// error, not unwind across the interface callback's C frame.
func TestReflectPanicInInterfaceCallbackIsContained(t *testing.T) {
	type point struct{ N int }

	L := NewState()
	defer L.Close()
	L.OpenLibs()

	// Non-pointer struct: reflect.ValueOf(iface).Elem() panics on any access.
	L.PushGoStruct(point{N: 1})
	L.SetGlobal("s")

	t.Run("index", func(t *testing.T) {
		err := L.DoString(`return s.N`)
		if err == nil {
			t.Fatal("expected error from reflect panic, got nil")
		}
		mustBeAlive(t, L)
	})

	t.Run("newindex", func(t *testing.T) {
		err := L.DoString(`s.N = 5`)
		if err == nil {
			t.Fatal("expected error from reflect panic, got nil")
		}
		mustBeAlive(t, L)
	})
}

// TestGoPanicThenLuaErrorDoesNotCorruptState mirrors _example/error.go, which
// SIGSEGVs on unfixed code: a Go panic unwound across the in-flight lua_pcall
// leaves the VM's call chain corrupted, and the NEXT plain Lua error — whose
// message handler walks the stack (StackTrace -> lua_getinfo) — crashes the
// process. With panics contained at the cgo boundary both errors report
// cleanly and the VM stays intact.
func TestGoPanicThenLuaErrorDoesNotCorruptState(t *testing.T) {
	L := NewState()
	defer L.Close()
	L.OpenLibs()

	L.Register("faultyfunc", func(l *State) int {
		panic(errors.New("An error"))
	})

	// Step 1 (testDefault): an ordinary Lua runtime error first — the full
	// example sequence matters, corruption paths are order-sensitive.
	if err := L.DoString(`print("Unknown variable" .. x)`); err == nil {
		t.Fatal("expected the concat error, got nil")
	}

	// Step 2 (testRegistered): a Go panic crosses the in-flight lua_pcall.
	if err := L.DoString(`faultyfunc()`); err == nil {
		t.Fatal("expected error from panicking Go function, got nil")
	}

	// Step 3 (test2): on unfixed code this dies with SIGSEGV in lua_getinfo.
	err := L.DoString(`error("Some error")`)
	if err == nil {
		t.Fatal("expected the Lua error, got nil")
	}
	if !strings.Contains(err.Error(), "Some error") {
		t.Errorf("expected the Lua error's own message, got: %v", err)
	}
	mustBeAlive(t, L)
}

// TestLuaRuntimeErrorKeepsStackTrace verifies that the message-handler path
// (no Go panic involved) still produces a LuaError with the trace captured
// while the erroring stack was alive.
func TestLuaRuntimeErrorKeepsStackTrace(t *testing.T) {
	L := NewState()
	defer L.Close()
	L.OpenLibs()

	err := L.DoString(`
		local function deep() error("deep-error") end
		local function outer() deep() end
		outer()
	`)
	if err == nil {
		t.Fatal("expected runtime error, got nil")
	}
	luaErr, ok := err.(*LuaError)
	if !ok {
		t.Fatalf("expected *LuaError, got %T: %v", err, err)
	}
	if !strings.Contains(luaErr.Error(), "deep-error") {
		t.Errorf("unexpected message: %q", luaErr.Error())
	}
	if len(luaErr.StackTrace()) == 0 {
		t.Error("expected non-empty stack trace captured by the message handler")
	}
	mustBeAlive(t, L)
}

// TestExecutionLimitStopsRunawayLoop verifies the hook path: the execution
// limit fires inside a running chunk, is raised as a lua_error from the C
// hook wrapper, and surfaces as a normal error with the VM intact.
//
// NOTE (LuaJIT): count hooks never fire inside JIT-compiled traces. A hot
// loop that reaches the trace-compile threshold before the hook interval
// elapses becomes unstoppable (empirically: `while true do end` with a limit
// of 10000 hangs forever). jit.off() keeps the chunk interpreted so the test
// deterministically exercises what this fix changes — WHAT happens when the
// hook fires — not WHETHER LuaJIT delivers it, which is out of scope here.
func TestExecutionLimitStopsRunawayLoop(t *testing.T) {
	L := NewState()
	defer L.Close()
	L.OpenLibs()

	L.SetExecutionLimit(10000)
	err := L.DoString(`if jit then jit.off() end while true do end`)
	if err == nil {
		t.Fatal("expected execution-limit error, got nil")
	}
	if !strings.Contains(err.Error(), ExecutionQuantumExceeded) {
		t.Errorf("expected quantum-exceeded error, got: %v", err)
	}
	L.SetHook(nil, 0)
	mustBeAlive(t, L)
}

// TestRepeatedErrorsDoNotOverflowCStack reproduces upstream issue #24: with
// the panicking message handler, every error through Call left lua_pcall's
// error handling unfinished; repeated erroring calls degraded to
// "error in error handling" (~120 iterations on PUC Lua 5.1 per the report;
// on LuaJIT/arm64 the second call already fails). With the handler no longer
// panicking, the VM unwinds each error completely.
func TestRepeatedErrorsDoNotOverflowCStack(t *testing.T) {
	L := NewState()
	defer L.Close()
	L.OpenLibs()

	if err := L.DoString(`function bad() error("something bad") end`); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 1000; i++ {
		L.GetGlobal("bad")
		err := L.Call(0, 0)
		if err == nil {
			t.Fatalf("iteration %d: expected error, got nil", i)
		}
		if strings.Contains(err.Error(), "error in error handling") {
			t.Fatalf("iteration %d: error handling exhausted (upstream issue #24): %v", i, err)
		}
		// The error must stay THE error on every iteration; in the broken
		// state the message degrades once error handling is wedged.
		if !strings.Contains(err.Error(), "something bad") {
			t.Fatalf("iteration %d: error degraded: %v", i, err)
		}
		// Discard the error value like any real consumer would (the original
		// #24 report restored the top between calls as well); the loop must
		// not depend on unbounded Lua stack growth.
		L.SetTop(0)
	}
}

// TestBackwardCompatibility pins the API contracts that existing Go-side
// consumers of this library rely on. The panic-containment fix changes HOW an
// error travels (through the VM as a lua_error instead of a Go panic across C
// frames) but MUST NOT change WHAT Call/DoString return. One subtest per
// contract:
//
//	C1: RaiseError        -> caller gets the original *LuaError back
//	C2: panic(customErr)  -> caller gets that exact error value back
//	C3: Lua runtime error -> LuaError.Code() stays LUA_ERRERR (historical)
//	C4: internal pocket   -> a stale recovered value never leaks into an
//	                         unrelated later error
func TestBackwardCompatibility(t *testing.T) {
	newVM := func(t *testing.T) *State {
		t.Helper()
		L := NewState()
		t.Cleanup(L.Close)
		L.OpenLibs()
		return L
	}

	t.Run("C1: RaiseError result is the original LuaError (code 0, message, trace)", func(t *testing.T) {
		// CONTRACT: err returned by Call/DoString for a RaiseError is the
		//           *LuaError built at the raise point: Code()==0 (NOT the
		//           lua_pcall status), the message exactly as RaiseError
		//           constructs it ("src:line: " position prefix + text — see
		//           RaiseError), the trace captured where it was called.
		// OLD PATH: RaiseError panicked with &LuaError{0, prefix+msg, trace};
		//           callEx recovered it and returned the object as-is.
		// SCENARIO: Go function calls RaiseError("original-object") →
		//           DoString returns *LuaError{code=0, "...: original-object",
		//           trace≠∅}.
		L := newVM(t)
		L.Register("raiser", func(l *State) int {
			l.RaiseError("original-object")
			return 0
		})

		err := L.DoString(`raiser()`)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		le, ok := err.(*LuaError)
		if !ok {
			t.Fatalf("expected *LuaError, got %T: %v", err, err)
		}
		if le.Code() != 0 {
			t.Errorf("Code() must stay 0 as RaiseError always produced, got %d", le.Code())
		}
		if !strings.HasSuffix(le.Error(), ": original-object") {
			t.Errorf("message must be RaiseError's own (position prefix + text), got %q", le.Error())
		}
		if len(le.StackTrace()) == 0 {
			t.Error("trace must be the one captured at the raise point, got empty")
		}
		mustBeAlive(t, L)
	})

	t.Run("C2: a custom error value panicked in Go is returned identically", func(t *testing.T) {
		// CONTRACT: panic(err) inside a Go callback surfaces from
		//           Call/DoString as that exact error value — errors.Is/As
		//           against caller-defined sentinels keeps working.
		// OLD PATH: the panic was recovered in callEx and, being an error,
		//           returned unchanged.
		// SCENARIO: Go function panics with a sentinel error → DoString's
		//           err satisfies errors.Is(err, sentinel).
		L := newVM(t)
		sentinel := errors.New("sentinel-error")
		L.Register("failer", func(l *State) int {
			panic(sentinel)
		})

		err := L.DoString(`failer()`)
		if !errors.Is(err, sentinel) {
			t.Errorf("errors.Is against the caller's sentinel must hold, got %T: %v", err, err)
		}
		mustBeAlive(t, L)
	})

	t.Run("C3: plain Lua runtime errors keep LuaError.Code()==LUA_ERRERR", func(t *testing.T) {
		// CONTRACT: LuaError.Code() for an ordinary script error stays
		//           LUA_ERRERR — the constant existing code compares against.
		// OLD PATH: the default message handler hardcoded LUA_ERRERR into
		//           every LuaError it built (semantically wrong — LUA_ERRERR
		//           means "error in error handling" — but it is the
		//           observable value consumers may depend on).
		// SCENARIO: script calls error("...") → DoString returns *LuaError
		//           with Code()==LUA_ERRERR.
		L := newVM(t)
		err := L.DoString(`error("plain-lua-error")`)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		le, ok := err.(*LuaError)
		if !ok {
			t.Fatalf("expected *LuaError, got %T: %v", err, err)
		}
		if le.Code() != LUA_ERRERR {
			t.Errorf("Code() must stay LUA_ERRERR (%d) as the old handler hardcoded, got %d", LUA_ERRERR, le.Code())
		}
		mustBeAlive(t, L)
	})

	t.Run("C4: stale recovered value never leaks into a later unrelated error", func(t *testing.T) {
		// CONTRACT: the pendingPanic pocket (the mechanism behind C1/C2) must
		//           not misattribute: if a Lua-level pcall consumes the
		//           contained Go error, a LATER unrelated error in the same
		//           Call is reported as itself, never as the stale original.
		// MECHANISM UNDER TEST: the message-mismatch guard in callEx — the
		//           pocket is used only when the error string on the stack
		//           equals the recorded one.
		// SCENARIO: unsafe_pcall catches a contained Go error (pocket goes
		//           stale) → the script then raises error("later-unrelated")
		//           → DoString's err carries "later-unrelated" and does NOT
		//           satisfy errors.Is against the stale sentinel.
		L := newVM(t)
		sentinel := errors.New("stale-sentinel")
		L.Register("failer", func(l *State) int {
			panic(sentinel)
		})

		err := L.DoString(`
			local ok, e = unsafe_pcall(failer)
			assert(ok == false, "pcall must catch the contained Go error")
			error("later-unrelated")
		`)
		if err == nil {
			t.Fatal("expected the later error, got nil")
		}
		if errors.Is(err, sentinel) {
			t.Errorf("stale recovered value misattributed to the later error: %v", err)
		}
		if !strings.Contains(err.Error(), "later-unrelated") {
			t.Errorf("expected the later error's own message, got: %v", err)
		}
		mustBeAlive(t, L)
	})
}
