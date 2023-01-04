package pkg1

type Impl1 struct{}

func NewImpl1(arg1 string, arg2 int) (*Impl1, error) {
	return &Impl1{}, nil
}
