package appleseed

type Param struct {
	Str  string
	X, Y int64
}

type Res struct {
	Str string
	Add int64
}

type Cal struct{}

func (c *Cal) Add(param *Param, res *Res) error {
	res.Add = param.X + param.Y
	res.Str = param.Str
	return nil
}
