// Copyright 2018 Marc-Antoine Ruel. All rights reserved.
// Use of this source code is governed under the Apache License, Version 2.0
// that can be found in the LICENSE file.

package stack_test

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/maruel/panicparse/v2/stack"
)

const crash = `panic: oh no!

goroutine 1 [running]:
panic(0x0, 0x0)
	/home/user/src/golang/src/runtime/panic.go:464 +0x3e6
main.crash2(0x7fe50b49d028, 0xc82000a1e0)
	/home/user/go/src/github.com/maruel/panicparse/cmd/pp/main.go:45 +0x23
main.main()
	/home/user/go/src/github.com/maruel/panicparse/cmd/pp/main.go:50 +0xa6
`

func Example() {
	// Optional: Check for GOTRACEBACK being set, in particular if there is only
	// one goroutine returned.
	in := bytes.NewBufferString(crash)
	c, err := stack.ParseDump(in, os.Stdout, true)
	if err != nil {
		return
	}

	// Find out similar goroutine traces and group them into buckets.
	buckets := stack.Aggregate(c.Goroutines, stack.AnyValue)

	// Calculate alignment.
	srcLen := 0
	pkgLen := 0
	for _, bucket := range buckets {
		for _, line := range bucket.Signature.Stack.Calls {
			if l := len(fmt.Sprintf("%s:%d", line.SrcName, line.Line)); l > srcLen {
				srcLen = l
			}
			if l := len(filepath.Base(line.Func.ImportPath)); l > pkgLen {
				pkgLen = l
			}
		}
	}

	for _, bucket := range buckets {
		// Print the goroutine header.
		extra := ""
		if s := bucket.SleepString(); s != "" {
			extra += " [" + s + "]"
		}
		if bucket.Locked {
			extra += " [locked]"
		}

		if len(bucket.CreatedBy.Calls) != 0 {
			extra += fmt.Sprintf(" [Created by %s.%s @ %s:%d]", bucket.CreatedBy.Calls[0].Func.DirName, bucket.CreatedBy.Calls[0].Func.Name, bucket.CreatedBy.Calls[0].SrcName, bucket.CreatedBy.Calls[0].Line)
		}
		fmt.Printf("%d: %s%s\n", len(bucket.IDs), bucket.State, extra)

		// Print the stack lines.
		for _, line := range bucket.Stack.Calls {
			fmt.Printf(
				"    %-*s %-*s %s(%s)\n",
				pkgLen, line.Func.DirName, srcLen,
				fmt.Sprintf("%s:%d", line.SrcName, line.Line),
				line.Func.Name, &line.Args)
		}
		if bucket.Stack.Elided {
			io.WriteString(os.Stdout, "    (...)\n")
		}
	}
	// Output:
	// panic: oh no!
	//
	// 1: running
	//          panic.go:464 panic(0, 0)
	//     main main.go:45   crash2(0x7fe50b49d028, 0xc82000a1e0)
	//     main main.go:50   main()
}
