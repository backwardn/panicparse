// Copyright 2018 Marc-Antoine Ruel. All rights reserved.
// Use of this source code is governed under the Apache License, Version 2.0
// that can be found in the LICENSE file.

package stack_test

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"runtime"

	"github.com/maruel/panicparse/stack"
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
			if l := len(line.SrcLine()); l > srcLen {
				srcLen = l
			}
			if l := len(line.Func.PkgName()); l > pkgLen {
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
		if c := bucket.CreatedByString(false); c != "" {
			extra += " [Created by " + c + "]"
		}
		fmt.Printf("%d: %s%s\n", len(bucket.IDs), bucket.State, extra)

		// Print the stack lines.
		for _, line := range bucket.Stack.Calls {
			fmt.Printf(
				"    %-*s %-*s %s(%s)\n",
				pkgLen, line.Func.PkgName(), srcLen, line.SrcLine(),
				line.Func.Name(), &line.Args)
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

func ExampleSnapshot() {
	// We don't know how big the buffer needs to be to collect
	// all the goroutines. Start with 1 MB and try a few times, doubling each time.
	// Give up and use a truncated trace if 64 MB is not enough.
	buf := make([]byte, 1<<20)
	for i := 0; ; i++ {
		n := runtime.Stack(buf, true)
		if n < len(buf) {
			buf = buf[:n]
			break
		}
		if len(buf) >= 64<<20 {
			// Filled 64 MB - stop there.
			break
		}
		buf = make([]byte, 2*len(buf))
	}
	c, err := ParseDump(bytes.NewReader(buf), ioutil.Discard, false)
	c.Print()
}

func writeGoroutineStacks(w io.Writer) error {
	_, err := w.Write(buf)
	return err
}

func ExampleSnapshot_HTTP() {
	// Make it similar to net/http/pprof, which calls into writeGoroutineStacks()
	// into runtime/pprof.
	p := func() {
	}
	http.HandleFunc("/debug/pprof/profile/goroutine", p)
	b := make([]byte, 1024*1024)
	_ = runtime.Stack(b, true)
	c, err := ParseDump(bytes.NewReader(s), ioutil.Discard, false)
	c.Print()
}
