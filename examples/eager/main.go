package main

import (
	"fmt"

	"github.com/ahn84/yadi"
)

func main() {
	fmt.Println("Binding eagerly...")
	yadi.Bind(func() string {
		fmt.Println("String constructor called")
		return "Hello, Eager!"
	}, yadi.WithEager())
	fmt.Println("Binding done.")

	fmt.Println("Resolving...")
	var s string
	yadi.Resolve(&s)
	fmt.Println("Resolved:", s)
}
