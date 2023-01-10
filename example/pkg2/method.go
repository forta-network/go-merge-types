package pkg2

type Int int

func (impl *Impl2) Foo(arg1 string, arg2 int, arg3 map[string]interface{}) (*Int, error) {
	var i Int = 12
	return &i, nil
}

func (impl *Impl2) SingleReturnVal(arg string) (int, error) {
	return 0, nil
}

func (impl *Impl2) NoReturnVal() error {
	return nil
}
