package main

import "github.com/ruitao/golua/lua"
import "unsafe"
import "fmt"

var refHolder [][]byte;


//a terrible allocator!
//meant to be illustrative of the mechanics,
//not usable as an actual implementation
func AllocatorF(ptr unsafe.Pointer, osize uint, nsize uint) unsafe.Pointer {
	if(nsize == 0) {
		//TODO: remove from reference holder
	} else if(osize != nsize) {
		//TODO: remove old ptr from list if its in there
		slice := make([]byte,nsize);
		ptr = unsafe.Pointer(&(slice[0]));
		//TODO: add slice to holder
		l := len(refHolder);
		refHolder = refHolder[0:l+1];
		refHolder[l] = slice;
	}
	//fmt.Println("in allocf");
	return ptr;
}


func A2(ptr unsafe.Pointer, osize uint, nsize uint) unsafe.Pointer {
	return AllocatorF(ptr,osize,nsize);
}

func main() {

	refHolder = make([][]byte,0,500);

	L := lua.NewStateAlloc(AllocatorF);
	defer L.Close()
	L.OpenLibs();

	L.SetAllocf(A2);

	for i:=0; i < 10; i++ {
		L.GetField(lua.LUA_GLOBALSINDEX, "print");
		L.PushString("Hello World!");
		L.Call(1,0);
	}

	fmt.Println(len(refHolder));
}
