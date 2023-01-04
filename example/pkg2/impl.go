package pkg2

type Impl2 struct{}

func NewImpl2(arg2 int64) (*Impl2, error) {
	return &Impl2{}, nil
}
