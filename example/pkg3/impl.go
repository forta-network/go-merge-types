package pkg3

import "sync"

type Impl3 struct{}

type Foo struct {
	Baz int
}

func NewImpl3(arg2 int, arg3 *sync.WaitGroup, arg4 *Foo) (*Impl3, error) {
	return &Impl3{}, nil
}
