package telegram

import "strings"

const (
	CommandTrackProduct = "/trackproduct"
	CommandListProducts = "/listproducts"
	CommandCancel       = "/cancel"
	CommandHelp         = "/help"
	CommandYes          = "/yes"
	CommandNo           = "/no"

	CommandPrefixPage          = "/page_"
	CommandPrefixDeleteProduct = "/del_"
)

type CommandsDictionary interface {
	GetCommands() []string
}

func IsCancelCommand(command string) bool {
	return command == CommandCancel
}

func IsHelpCommand(command string) bool {
	return command == CommandHelp || isCommandInDictionary(command, &HelpDictionary{})
}

func IsWelcomeCommand(command string) bool {
	return isCommandInDictionary(command, &WelcomeDictionary{})
}

func IsTrackProductCommand(command string) bool {
	return command == CommandTrackProduct
}

func IsListProductsCommand(command string) bool {
	return command == CommandListProducts
}

func IsDeleteProductCommand(command string) bool {
	return strings.HasPrefix(command, CommandPrefixDeleteProduct)
}

func isCommandInDictionary(command string, dictionary CommandsDictionary) bool {
	command = strings.ToLower(strings.TrimSpace(command))

	for _, dictionaryCommand := range dictionary.GetCommands() {
		if dictionaryCommand != command {
			continue
		}

		return true
	}

	return false
}
