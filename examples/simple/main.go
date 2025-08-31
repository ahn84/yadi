package main

import (
	"fmt"

	"github.com/ahn84/yadi"
)

type Greeter interface {
	Greet() string
}

type SimpleGreeter struct{}

func (g *SimpleGreeter) Greet() string {
	return "Hello, World!"
}

func main() {
	yadi.Bind(func() Greeter {
		return &SimpleGreeter{}
	})

	var greeter Greeter
	yadi.Resolve(&greeter)

	fmt.Println(greeter.Greet())
}
