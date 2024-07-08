package telegram

type HelpDictionary struct{}

func (d *HelpDictionary) GetCommands() []string {
	return []string{
		CommandHelp,
		"help",
		"helb",
		"хелп",
		"хелб",
		"/commands",
		"commands",
		"команды",
		"sos",
		"/помощь",
		"помощь",
		"помогите",
	}
}

type WelcomeDictionary struct{}

func (d *WelcomeDictionary) GetCommands() []string {
	return []string{
		"/hi",
		"hi",
		"/hello",
		"hello",
		"привет",
		"прив",
		"хай",
		"дороу",
		"дратути",
	}
}
