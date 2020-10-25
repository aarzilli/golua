package main

import "github.com/aarzilli/golua/lua"
import "unsafe"
import "fmt"

var refHolder = map[unsafe.Pointer][]byte{}

//a terrible allocator!
//meant to be illustrative of the mechanics,
//not usable as an actual implementation
func AllocatorF(ptr unsafe.Pointer, osize uint, nsize uint) unsafe.Pointer {
	if nsize == 0 {
		if _, ok := refHolder[ptr]; ok {
			delete(refHolder, ptr)
		}
		ptr = unsafe.Pointer(nil)
	} else if osize != nsize {
		slice := make([]byte, nsize)

		if oldslice, ok := refHolder[ptr]; ok {
			copy(slice, oldslice)
			_ = oldslice
			delete(refHolder, ptr)
		}

		ptr = unsafe.Pointer(&(slice[0]))
		refHolder[ptr] = slice
	}
	//fmt.Println("in allocf");
	return ptr
}

func A2(ptr unsafe.Pointer, osize uint, nsize uint) unsafe.Pointer {
	return AllocatorF(ptr, osize, nsize)
}

func main() {

	//refHolder = make([][]byte,0,500);

	L := lua.NewStateAlloc(&AllocatorF)
	defer L.Close()
	L.OpenLibs()

	L.SetAllocf(&A2)

	for i := 0; i < 10; i++ {
		L.GetGlobal("print")
		L.PushString("Hello World!")
		L.Call(1, 0)
	}

	fmt.Println(len(refHolder))
}
