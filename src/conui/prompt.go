package conui

type PromptAction struct {
	Message string
	Action  func()
}
