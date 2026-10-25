package main

import (
	"fmt"

	"github.com/ahn84/yadi"
)

func main() {
	fmt.Println("Binding eagerly...")
	di.Bind(func() string {
		fmt.Println("String constructor called")
		return "Hello, Eager!"
	}, di.WithEager())
	fmt.Println("Binding done.")

	fmt.Println("Resolving...")
	var s string
	di.Resolve(&s)
	fmt.Println("Resolved:", s)
}
