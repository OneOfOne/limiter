# limiter [![GoDoc](http://godoc.org/github.com/OneOfOne/limiter?status.svg)](http://godoc.org/github.com/OneOfOne/limiter) [![Build Status](https://travis-ci.org/OneOfOne/limiter.svg?branch=master)](https://travis-ci.org/OneOfOne/limiter)

limiter provides a simple limited job queue with context support

## ack Example

```go
package main

import (
	"context"

	"github.com/OneOfOne/limiter"
)

func main() {
	l := limiter.NewWithContext(context.Background(), runtime.NumCPU())

	for i := range [1000]struct{}{} {
		l.Do(func(ctx context.Context) error {
			// do something
		})
	}
	l.Wait()
}
```

## License

This project is released under the [BSD 3-clause "New" or "Revised" License](https://github.com/golang/go/blob/master/LICENSE).
