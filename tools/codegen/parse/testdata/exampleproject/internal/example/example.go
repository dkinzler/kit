package example

import (
	"context"
)

type X struct {
	A []string
	B map[string]int
}

type Y struct {
	C float64
	D map[string][]int
}

type ExampleInterface interface {
	Method1(ctx context.Context, a string, x X) (Y, error)
	Method2(m map[int][]map[string]int) error
	Method3()
}
