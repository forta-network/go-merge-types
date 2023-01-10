package pkg3

import (
	"math/big"
	biggie "math/big" // complicating import
)

type Result3 struct {
	B int
	C *biggie.Int
}

func (impl *Impl3) Foo(arg2 int, arg3 *biggie.Int) (*big.Int, error) {
	return big.NewInt(13), nil
}

type Something struct {
	ABC string
}

func (impl *Impl3) Bar(arg1 map[string]interface{}) error {
	return nil
}

func (impl *Impl3) ArrayMethod(sli []*Something, arr [32]*Something) error {
	return nil
}

func (impl *Impl3) ChanMethod(chan1 chan *Something, chan2 <-chan *Something, chan3 chan<- *Something) error {
	return nil
}

func (impl *Impl3) MapMethod(m map[string]*Something) error {
	return nil
}

func (impl *Impl3) FooBarBaz() {}

func (impl *Impl3) NoReturnVal(arg int) error {
	return nil
}
