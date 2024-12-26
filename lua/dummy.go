//go:build dummy
// +build dummy

// This file is part of a workaround for `go mod vendor` which won't
// vendor C files if there are no Go files in the same directory.
// This prevents the C header files in lua/ from being vendored.
//
// This Go file imports the lua package where there is another
// dummy.go file which is the second part of this workaround.
//
// These two files combined make it so `go mod vendor` behaves correctly.
//
// See this issue for reference: https://github.com/golang/go/issues/26366

package lua

import (
	_ "github.com/aarzilli/golua/lua/lua51"
	_ "github.com/aarzilli/golua/lua/lua52"
	_ "github.com/aarzilli/golua/lua/lua53"
	_ "github.com/aarzilli/golua/lua/lua54"
)
