package main

import "github.com/ruitao/golua/lua"
import "fmt"

func test(L *lua.State) int {
	fmt.Println("hello world! from go!");
	return 0;
}

func test2(L *lua.State) int {
	arg := lua.CheckInteger(L,-1);
	argfrombottom := lua.CheckInteger(L,1);
	fmt.Print("test2 arg: ");
	fmt.Println(arg);
	fmt.Print("from bottom: ");
	fmt.Println(argfrombottom);
	return 0;
}

func main() {
	var L *lua.State;

	L = lua.NewState();
	L.OpenLibs();

	L.GetField(lua.LUA_GLOBALSINDEX, "print");
	L.PushString("Hello World!");
	L.Call(1,0);

	L.PushGoFunction(test);
	L.PushGoFunction(test);
	L.PushGoFunction(test);
	L.PushGoFunction(test);

	L.PushGoFunction(test2);
	L.PushInteger(42);
	L.Call(1,0);


	L.Call(0,0);
	L.Call(0,0);
	L.Call(0,0);

	L.Close();
}
