package telegram

import (
	"bot/internal/app/helpers"
	"bot/internal/app/logger"
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"
)

type Emoji string

const (
	EmojiOkHand            Emoji = "👌"
	EmojiHighVoltage       Emoji = "⚡"
	EmojiThumbsUp          Emoji = "👍"
	EmojiNeutralFace       Emoji = "😐"
	EmojiMoneyMouthFace    Emoji = "🤑"
	EmojiParty             Emoji = "🥳"
	EmojiWhiteFrowningFace Emoji = "☹️"
	EmojiWhiteCheckMark    Emoji = "✅"
	EmojiX                 Emoji = "❌"
)

type UrlParams interface {
	ToString() string
}

type RequestData interface {
	ToJson() ([]byte, error)
}

type Response struct {
	Ok          bool            `json:"ok"`
	Result      json.RawMessage `json:"result,omitempty"`
	ErrorCode   int             `json:"error_code,omitempty"`
	Description string          `json:"description,omitempty"`
}

type Bot struct {
	apiEndpoint string
	token       string
	WhoAmI      BotUser
	logger      logger.LoggerInterface
}

// Constructor.
func NewBot(token string, logger logger.LoggerInterface) (Bot, error) {
	bot := Bot{
		apiEndpoint: "https://api.telegram.org/bot<token>/<method>",
		token:       token,
		logger:      logger,
	}

	whoAmI, err := bot.getMe()

	if err != nil {
		return Bot{}, err
	}

	bot.WhoAmI = whoAmI

	return bot, nil
}

// Get basic information about the bot.
// https://core.telegram.org/bots/api#getme
func (b *Bot) getMe() (BotUser, error) {
	var result BotUser

	endpoint := b.getEndpoint("getMe", nil)
	response, err := b.sendRequest(endpoint, nil, false)
	if err != nil {
		return result, err
	}

	if err := json.Unmarshal(response.Result, &result); err != nil {
		return result, err
	}

	return result, nil
}

// Get incoming updates.
// https://core.telegram.org/bots/api#getupdates
func (b *Bot) getUpdates(offset int) ([]Update, error) {
	var result []Update

	endpoint := b.getEndpoint("getUpdates", &GetUpdatesParams{
		Offset:  offset,
		Timeout: 20,
	})

	response, err := b.sendRequest(endpoint, nil, true)

	if err != nil {
		return result, err
	}

	if err := json.Unmarshal(response.Result, &result); err != nil {
		return result, err
	}

	return result, nil
}

// Listen for incoming updates and apply a callback function to each item.
func (b *Bot) ListenForUpdates(callback func(update Update), updateIdOffset int) {
	updatesChannel := make(chan Update)

	go func() {
		for {
			updates, err := b.getUpdates(updateIdOffset)

			if err != nil {
				b.logger.Println("Failed to get updates, retrying in 3 seconds...", err)

				time.Sleep(time.Second * 10)

				continue
			}

			for _, update := range updates {
				if update.UpdateId < updateIdOffset {
					continue
				}

				updateIdOffset = update.UpdateId + 1
				updatesChannel <- update
			}
		}
	}()

	for update := range updatesChannel {
		callback(update)
	}
}

// Send text message.
// https://core.telegram.org/bots/api#sendmessage
func (b *Bot) SendMessage(toChatId int, request SendMessageRequest) (Message, error) {
	var result Message

	endpoint := b.getEndpoint("sendMessage", &SendMessageParams{
		ChatId: toChatId,
	})

	response, err := b.sendRequest(endpoint, &request, false)

	if err != nil {
		return result, err
	}

	if err := json.Unmarshal(response.Result, &result); err != nil {
		return result, err
	}

	return result, nil
}

// Answer to callback query.
// https://core.telegram.org/bots/api#answercallbackquery
func (b *Bot) AnswerCallbackQuery(callbackQueryId string) {
	endpoint := b.getEndpoint("answerCallbackQuery", &AnswerCallbackQueryParams{
		CallbackQueryId: callbackQueryId,
	})

	b.sendRequest(endpoint, nil, false)
}

// Edit message.
// https://core.telegram.org/bots/api#editmessagetext
func (b *Bot) EditMessage(chatId int, messageId int, request EditMessageRequest) {
	endpoint := b.getEndpoint("editMessageText", &EditMessageParams{
		ChatId:    chatId,
		MessageId: messageId,
	})

	b.sendRequest(endpoint, &request, false)
}

func (b *Bot) SetMessageReaction(toChatId int, toMessageId int, emoji Emoji) {
	endpoint := b.getEndpoint("setMessageReaction", &SetMessageReactionParams{
		ChatId:    toChatId,
		MessageId: toMessageId,
	})

	request := SetMessageReactionRequest{
		Emoji: emoji,
	}

	b.sendRequest(endpoint, &request, false)
}

// Get method endpoint with optional URL parameters.
func (b *Bot) getEndpoint(method string, params UrlParams) string {
	endpoint := strings.Replace(b.apiEndpoint, "<method>", method, 1)

	if params != nil {
		endpoint = helpers.ConcatStrings(endpoint, "?", params.ToString())
	}

	return endpoint
}

// Send request to endpoint with optional data.
func (b *Bot) sendRequest(endpoint string, data RequestData, skipLogMessage bool) (Response, error) {
	var httpMethod string

	body := bytes.NewBuffer(nil)

	if data != nil {
		httpMethod = http.MethodPost

		jsonData, err := data.ToJson()
		if err != nil {
			b.logger.Println(err)
			return Response{}, err
		}

		body.Write(jsonData)

		if !skipLogMessage {
			b.logger.Println("Sending POST request to", endpoint, "with data", body)
		}
	} else {
		httpMethod = http.MethodGet

		if !skipLogMessage {
			b.logger.Println("Sending GET request to", endpoint)
		}
	}

	// don't expose token in logs
	endpoint = strings.Replace(endpoint, "<token>", b.token, 1)

	request, err := http.NewRequest(httpMethod, endpoint, body)
	if err != nil {
		b.logger.Println(err)
		return Response{}, err
	}

	request.Header.Set("Content-Type", "application/json")

	client := http.Client{}

	response, err := client.Do(request)
	if err != nil {
		b.logger.Println(err)
		return Response{}, err
	}

	defer response.Body.Close()

	return b.decodeResponse(response.Body)
}

// Decode response to generic struct.
func (b *Bot) decodeResponse(data io.Reader) (Response, error) {
	var responseDecoded Response

	decoder := json.NewDecoder(data)
	err := decoder.Decode(&responseDecoded)

	if err != nil {
		b.logger.Println(err)
		return Response{}, err
	}

	if !responseDecoded.Ok {
		errorMessage := helpers.ConcatStrings(strconv.Itoa(responseDecoded.ErrorCode), ": ", responseDecoded.Description)
		return Response{}, errors.New(errorMessage)
	}

	return responseDecoded, nil
}
