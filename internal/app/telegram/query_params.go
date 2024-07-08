package telegram

import (
	"net/url"
	"strconv"
)

// Query parameters for "getUpdates" method.
// https://core.telegram.org/bots/api#getupdates
type GetUpdatesParams struct {
	Offset  int
	Timeout int
}

func (params *GetUpdatesParams) ToString() string {
	data := make(url.Values)

	data.Add("offset", strconv.Itoa(params.Offset))
	data.Add("timeout", strconv.Itoa(params.Timeout))

	return data.Encode()
}

// Query parameters for "sendMessage" method.
// https://core.telegram.org/bots/api#sendmessage
type SendMessageParams struct {
	ChatId int
}

func (p *SendMessageParams) ToString() string {
	data := make(url.Values)

	data.Add("chat_id", strconv.Itoa(p.ChatId))

	return data.Encode()
}

// Query parameters for "answerCallbackQuery" method.
// https://core.telegram.org/bots/api#answercallbackquery
type AnswerCallbackQueryParams struct {
	CallbackQueryId string
}

func (p *AnswerCallbackQueryParams) ToString() string {
	data := make(url.Values)

	data.Add("callback_query_id", p.CallbackQueryId)

	return data.Encode()
}

// Query parameters for "editMessageText" method.
// https://core.telegram.org/bots/api#editmessagetext
type EditMessageParams struct {
	ChatId    int
	MessageId int
}

func (p *EditMessageParams) ToString() string {
	data := make(url.Values)

	data.Add("chat_id", strconv.Itoa(p.ChatId))
	data.Add("message_id", strconv.Itoa(p.MessageId))

	return data.Encode()
}

// Query parameters for "setMessageReaction" method.
// https://core.telegram.org/bots/api#setmessagereaction
type SetMessageReactionParams struct {
	ChatId    int
	MessageId int
}

func (p *SetMessageReactionParams) ToString() string {
	data := make(url.Values)

	data.Add("chat_id", strconv.Itoa(p.ChatId))
	data.Add("message_id", strconv.Itoa(p.MessageId))

	return data.Encode()
}
