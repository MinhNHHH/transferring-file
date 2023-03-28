package message

type Wrapper struct {
	Type string
	Data interface{}
	From string
	To   string
}
