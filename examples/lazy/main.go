package main

import (
	"fmt"

	"github.com/ahn84/yadi"
)

type Service1 struct {
	Service2 yadi.Lazy[Service2]
}

func (s *Service1) Do() {
	fmt.Println("Service1 doing work")
	s2, _ := s.Service2.Resolve()
	s2.Do()
}

type Service2 struct {
	Service1 *Service1
}

func (s *Service2) Do() {
	fmt.Println("Service2 doing work")
}

func main() {
	yadi.Bind(func(s2 yadi.Lazy[Service2]) *Service1 {
		return &Service1{Service2: s2}
	})
	yadi.Bind(func(s1 *Service1) *Service2 {
		return &Service2{Service1: s1}
	})

	var s1 *Service1
	yadi.Resolve(&s1)

	s1.Do()
}
