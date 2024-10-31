package lua

import "fmt"

func (L *State) PrintStack() {
	t := L.GetTop()
	fmt.Printf("~ | TOP\n")
	for i := t; i >= 1; i-- {
		if L.IsBoolean(i) {
			fmt.Printf("%d | BOOL : %t\n", i, L.ToBoolean(i))
			continue
		}
		if L.IsNumber(i) {
			fmt.Printf("%d | NUM  : %f\n", i, L.ToNumber(i))
			continue
		}
		if L.IsString(i) {
			fmt.Printf("%d | STR  : %s\n", i, L.ToString(i))
			continue
		}
		if L.IsTable(i) {
			fmt.Printf("%d | TBL  : Size:%d\n", i, L.ObjLen(i))
			continue
		}
		if L.IsFunction(i) || L.IsGoFunction(i) {
			fmt.Printf("%d | FUNC\n", i)
			continue
		}
		if L.IsNil(i) {
			fmt.Printf("%d | NIL\n", i)
		}
	}
	fmt.Printf("~ | BOTTOM\n")
}
