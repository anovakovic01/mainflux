package api

type connectRes struct {
	err error
}

type subscribeRes struct {
	id        uint16
	topicsNum int
	err       error
}
