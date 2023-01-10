package pkg1

type Result1 struct {
	A string
	B float32
}

func (impl *Impl1) Foo(arg1 string) (*Result1, error) {
	return &Result1{
		A: arg1,
	}, nil
}

func (impl *Impl1) Bar(arg1 chan *string) {}

func (impl *Impl1) SingleReturnVal() (int, error) {
	return 0, nil
}
