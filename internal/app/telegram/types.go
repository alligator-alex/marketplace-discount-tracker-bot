package telegram

// https://core.telegram.org/bots/api#user
type User struct {
	Id        int    `json:"id"`
	IsBot     bool   `json:"is_bot"`
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
	UserName  string `json:"username"`
}

// https://core.telegram.org/bots/api#user
type BotUser struct {
	User
	CanJoinGroups           bool `json:"can_join_groups"`
	CanReadAllGroupMessages bool `json:"can_read_all_group_messages"`
	SupportsInlineQueries   bool `json:"supports_inline_queries"`
}

// https://core.telegram.org/bots/api#chat
type Chat struct {
	Id        int    `json:"id"`
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
	UserName  string `json:"username"`
	Type      string `json:"private"`
}

// https://core.telegram.org/bots/api#message
type Message struct {
	MessageId int    `json:"message_id"`
	From      User   `json:"from"`
	Chat      Chat   `json:"chat"`
	Date      int    `json:"date"`
	Text      string `json:"text"`
}

// https://core.telegram.org/bots/api#update
type Update struct {
	UpdateId      int           `json:"update_id"`
	Message       Message       `json:"message"`
	CallbackQuery CallbackQuery `json:"callback_query"`
}

// https://core.telegram.org/bots/api#callbackquery
type CallbackQuery struct {
	Id           string  `json:"id"`
	From         User    `json:"from"`
	Message      Message `json:"message"`
	ChatInstance string  `json:"chat_instance"`
	Data         string  `json:"data"`
}

// https://core.telegram.org/bots/api#inlinekeyboardmarkup
type InlineKeyboardMarkup struct {
	Keyboard [][]InlineKeyboardButton `json:"inline_keyboard"`
}

// https://core.telegram.org/bots/api#inlinekeyboardbutton
type InlineKeyboardButton struct {
	Text         string `json:"text"`
	CallbackData string `json:"callback_data"`
}
