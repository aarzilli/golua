package main

import (
	"strconv"
	"sync"

	"github.com/aarzilli/golua/lua"
)

func test(L *lua.State) int {
	println("hello from thread: " + strconv.Itoa(L.ToInteger(-1)))
	return 0
}

var wg sync.WaitGroup

func main() {
	L := lua.NewState()
	defer L.Close()
	L.OpenLibs()

	L.PushGoFunction(test)
	L.SetGlobal("test")

	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			thread := L.NewThread()

			thread.GetGlobal("test")
			thread.PushInteger(int64(i))
			if err := thread.Call(1, 0); err != nil {
				println("pain")
				println(err)
			}
		}()
	}

	wg.Wait()
}
