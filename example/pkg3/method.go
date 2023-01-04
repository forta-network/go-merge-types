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

func (impl *Impl3) Bar(arg1 map[string]interface{}) error {
	return nil
}
