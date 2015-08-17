package conui

type Prompt struct {
	Message string
	Action  func()
}
