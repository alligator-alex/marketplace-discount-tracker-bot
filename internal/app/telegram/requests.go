package telegram

import "encoding/json"

const parseModeHtml = "HTML"

type JsonObject map[string]any

// Request data for "sendMessage" method.
// https://core.telegram.org/bots/api#sendmessage
type SendMessageRequest struct {
	ReplyToMessageId   int
	Text               string
	ReplyMarkup        InlineKeyboardMarkup
	LinkPreviewOptions LinkPreviewOptions
}

func (r *SendMessageRequest) ToJson() ([]byte, error) {
	data := JsonObject{
		"text":                 r.Text,
		"parse_mode":           parseModeHtml,
		"link_preview_options": r.LinkPreviewOptions,
	}

	if r.ReplyToMessageId > 0 {
		data["reply_parameters"] = JsonObject{
			"message_id": r.ReplyToMessageId,
		}
	}

	if len(r.ReplyMarkup.Keyboard) > 0 {
		data["reply_markup"] = r.ReplyMarkup
	}

	return json.Marshal(data)
}

// Request data for "editMessageText" method.
// https://core.telegram.org/bots/api#editmessagetext
type EditMessageRequest struct {
	Text               string
	ReplyMarkup        InlineKeyboardMarkup
	LinkPreviewOptions LinkPreviewOptions
}

func (r *EditMessageRequest) ToJson() ([]byte, error) {
	data := JsonObject{
		"text":                 r.Text,
		"parse_mode":           parseModeHtml,
		"link_preview_options": r.LinkPreviewOptions,
	}

	if len(r.ReplyMarkup.Keyboard) > 0 {
		data["reply_markup"] = r.ReplyMarkup
	}

	return json.Marshal(data)
}

// Request data for "setMessageReaction" method.
// https://core.telegram.org/bots/api#setmessagereaction
type SetMessageReactionRequest struct {
	Emoji Emoji
}

func (r *SetMessageReactionRequest) ToJson() ([]byte, error) {
	data := JsonObject{
		"reaction": []JsonObject{
			{
				"type":  "emoji",
				"emoji": r.Emoji,
			},
		},
	}

	return json.Marshal(data)
}
