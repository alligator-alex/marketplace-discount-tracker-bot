package core

type Model struct {
	Id int
}

func (model Model) Exists() bool {
	return model.Id > 0
}
