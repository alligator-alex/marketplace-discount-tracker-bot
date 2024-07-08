package app

import (
	"bot/internal/app/core"
	"bot/internal/app/database"
	"bot/internal/app/helpers"
	"bot/internal/app/logger"
	"bot/internal/app/marketplace"
	"bot/internal/app/statemachine"
	"bot/internal/app/telegram"
	"crypto/md5"
	"encoding/hex"
	"log"
	"os"
	"reflect"
	"strconv"
	"strings"
	"time"
)

type TrackedProduct struct {
	scrapedAt      time.Time
	telegramChatId int
	telegramUserId int
	marketplace    marketplace.Marketplace
	url            string
	title          string
	thresholdPrice int
	currentPrice   int
	outOfStock     bool
}

func (p *TrackedProduct) GetScrapedAt() time.Time {
	return p.scrapedAt
}

func (p *TrackedProduct) GetSlug() string {
	return ""
}

func (p *TrackedProduct) GetTelegramChatId() int {
	return p.telegramChatId
}

func (p *TrackedProduct) GetTelegramUserId() int {
	return p.telegramUserId
}

func (p *TrackedProduct) GetUrl() string {
	return p.url
}

func (p *TrackedProduct) GetMarketplace() marketplace.Marketplace {
	return p.marketplace
}

func (p *TrackedProduct) IsActive() bool {
	return true
}

func (p *TrackedProduct) GetTitle() string {
	return p.title
}

func (p *TrackedProduct) GetThresholdPrice() int {
	return p.thresholdPrice
}

func (p *TrackedProduct) GetCurrentPrice() int {
	return p.currentPrice
}

func (p *TrackedProduct) IsOutOfStock() bool {
	return p.outOfStock
}

type TelegramBotApp struct {
	bot                telegram.Bot
	conversations      map[string]*telegram.Conversation
	markerplaceService marketplace.Service
	logger             logger.LoggerInterface
	timeLocation       *time.Location
}

func NewTelegramBotApp(token string, logger logger.LoggerInterface) TelegramBotApp {
	bot, err := telegram.NewBot(token, logger)
	if err != nil {
		log.Fatalln(err)
	}

	db, err := database.NewPostgres()
	if err != nil {
		log.Fatalln(err)
	}

	repository := marketplace.NewPostgresRepository(db, logger)

	timezone := os.Getenv("TIMEZONE")
	timeLocation, _ := time.LoadLocation(timezone)

	return TelegramBotApp{
		bot:                bot,
		conversations:      make(map[string]*telegram.Conversation),
		markerplaceService: marketplace.NewService(&repository, logger),
		logger:             logger,
		timeLocation:       timeLocation,
	}
}

func (app *TelegramBotApp) Run() {
	app.collectGarbage()
	app.watchTrackedProducts()

	app.logger.Println(helpers.ConcatStrings("I'm the @", app.bot.WhoAmI.UserName, " now"))

	defaultOffsetId := 0

	app.bot.ListenForUpdates(func(update telegram.Update) {
		var message telegram.Message

		if update.CallbackQuery.Id != "" {
			message = update.CallbackQuery.Message

			// mimic a message from user, not from bot
			message.From = update.CallbackQuery.From
			message.Text = update.CallbackQuery.Data
		} else {
			message = update.Message
		}

		hash := app.calculateConversationHash(message)

		conversation, exists := app.conversations[hash]

		if !exists {
			app.conversations[hash] = telegram.NewConversation(message.Chat.Id, message.From)

			conversation = app.conversations[hash]
		}

		conversation.LastMessage = message
		conversation.LastCallbackQueryId = update.CallbackQuery.Id

		app.processConversation(conversation)
	}, defaultOffsetId)
}

// Collect garbage (delete hanged conversations).
func (app *TelegramBotApp) collectGarbage() {
	const intervalInMinutes = 10

	gcTicker := time.NewTicker(intervalInMinutes * time.Minute)

	go func() {
		for range gcTicker.C {
			currentTimestamp := int(time.Now().Unix())

			for hash, conversation := range app.conversations {
				conversationTimestampHanged := conversation.LastMessage.Date + intervalInMinutes*60

				if conversationTimestampHanged > currentTimestamp {
					continue
				}

				conversation.Reset()
				delete(app.conversations, hash)
			}
		}
	}()
}

// Scrape tracked products in background.
func (app *TelegramBotApp) watchTrackedProducts() {
	watcher := marketplace.NewWatcher(app.markerplaceService, app.logger)

	resultChannel := make(chan marketplace.WatcherResult)

	go func() {
		for {
			err := watcher.Run(resultChannel)
			if err != nil {
				app.logger.Println("Error while watching:", err)
			}

			time.Sleep(marketplace.WatcherIntervalInMinutes * time.Minute)
		}
	}()

	go func() {
		for result := range resultChannel {
			request := telegram.SendMessageRequest{}

			if result.Scraped.IsOutOfStock() {
				continue
			}

			if result.Original.IsOutOfStock() && !result.Scraped.IsOutOfStock() {
				// in stock again
				request.Text = helpers.ConcatStrings(
					"Товар <b>«<a href=\"", result.Original.GetUrl(), "\">", result.Original.GetTitle(), "</a>»</b> (", marketplace.GetMarketplaceName(result.Original), ")",
					" снова в продаже!\n",
					"<b>Текущая цена: ", helpers.CurrencyFormat(helpers.CurrencyToMajor(result.Scraped.GetCurrentPrice())), "</b>",
				)
			} else if result.Original.GetThresholdPrice() > result.Scraped.GetCurrentPrice() {
				// price is now lower
				request.Text = helpers.ConcatStrings(
					"Цена на товар <b>«<a href=\"", result.Original.GetUrl(), "\">", result.Original.GetTitle(), "</a>»</b> (", marketplace.GetMarketplaceName(result.Original), ")",
					" снизилась!\n",
					"<b>Текущая цена: ", helpers.CurrencyFormat(helpers.CurrencyToMajor(result.Scraped.GetCurrentPrice())), "</b>",
				)
			}

			if request.Text == "" {
				continue
			}

			_, err := app.bot.SendMessage(result.Original.GetTelegramChatId(), request)
			if err != nil {
				app.logger.Println("ERROR! Unable to send listing message:", err)
				return
			}
		}
	}()
}

// Calculate conversation hash based on chat and user ids.
func (app *TelegramBotApp) calculateConversationHash(message telegram.Message) string {
	data := helpers.ConcatStrings(strconv.Itoa(message.Chat.Id), "_", strconv.Itoa(message.From.Id))
	hash := md5.Sum([]byte(data))

	return hex.EncodeToString(hash[:])
}

// Process conversation with user.
func (app *TelegramBotApp) processConversation(conversation *telegram.Conversation) {
	// "cancel" command
	if telegram.IsCancelCommand(conversation.LastMessage.Text) {
		conversation.Reset()
		app.bot.SetMessageReaction(conversation.ChatId, conversation.LastMessage.MessageId, telegram.EmojiOkHand)
		return
	}

	// "help" command
	if telegram.IsHelpCommand(conversation.LastMessage.Text) {
		request := telegram.SendMessageRequest{
			ReplyToMessageId: conversation.LastMessage.MessageId,
			Text:             "Список всех команд доступен в меню",
		}

		app.bot.SendMessage(conversation.ChatId, request)
		return
	}

	// "welcome" command
	if telegram.IsWelcomeCommand(conversation.LastMessage.Text) {
		request := telegram.SendMessageRequest{
			ReplyToMessageId: conversation.LastMessage.MessageId,
			Text:             helpers.ConcatStrings("Hello there, @", conversation.User.UserName),
		}

		app.bot.SendMessage(conversation.ChatId, request)
		return
	}

	// "track product" command
	if telegram.IsTrackProductCommand(conversation.LastMessage.Text) {
		conversation.StateMachine = marketplace.NewFsm()

		_, err := conversation.StateMachine.TriggerEvent(marketplace.EventAskForUrl)
		if err != nil {
			app.logErrorAndSendMessage(
				conversation,
				err,
				helpers.ConcatStrings("Unable to trigger state machine \"", string(marketplace.EventAskForUrl), "\" event"),
				"Не могу перейти к запросу ссылки",
			)
			return
		}
	}

	// "list products" command
	if telegram.IsListProductsCommand(conversation.LastMessage.Text) {
		conversation.StateMachine = marketplace.NewFsm()

		_, err := conversation.StateMachine.TriggerEvent(marketplace.EventList)
		if err != nil {
			app.logErrorAndSendMessage(
				conversation,
				err,
				helpers.ConcatStrings("Unable to trigger state machine \"", string(marketplace.EventList), "\" event"),
				"Не могу перейти к отображению списка",
			)
			return
		}
	}

	// "delete product" command
	if telegram.IsDeleteProductCommand(conversation.LastMessage.Text) {
		if !conversation.StateMachine.IsInitialized() {
			conversation.StateMachine = marketplace.NewFsm()
			conversation.StateMachine.TriggerEvent(marketplace.EventList)
		}

		_, err := conversation.StateMachine.TriggerEvent(marketplace.EventDelete)
		if err != nil {
			app.logErrorAndSendMessage(
				conversation,
				err,
				helpers.ConcatStrings("Unable to trigger state machine \"", string(marketplace.EventDelete), "\" event"),
				"Не могу перейти к удалению",
			)
			return
		}
	}

	// check state machine after receiving a command
	if conversation.StateMachine.GetCurrentState() != statemachine.StateIdle {
		app.processStateMachine(conversation)
		return
	}

	// unknown command
	app.bot.SendMessage(conversation.ChatId, telegram.SendMessageRequest{
		ReplyToMessageId: conversation.LastMessage.MessageId,
		Text:             "I'm sorry, Dave. I'm afraid I can't do that.",
	})
}

// Process conversation's state machine.
func (app *TelegramBotApp) processStateMachine(conversation *telegram.Conversation) {
	if !app.isTrackedProductContext(conversation) && conversation.StateMachine.IsInOneOfStates([]statemachine.State{
		marketplace.StateAskingForUrl,
		marketplace.StateWaitingForUrl,
		marketplace.StateScraping,
	}) {
		conversation.StoreContext(telegram.ConversationCtxProduct, TrackedProduct{
			telegramChatId: conversation.ChatId,
			telegramUserId: conversation.User.Id,
		})
	}

	switch conversation.StateMachine.GetCurrentState() {
	case marketplace.StateAskingForUrl:
		app.askForMarketplaceUrl(conversation)
	case marketplace.StateWaitingForUrl:
		app.waitForMarketplaceUrl(conversation)
	case marketplace.StateScraping:
		app.scrapeMarketplaceUrl(conversation)
	case marketplace.StateListing:
		app.showMarketplaceListing(conversation)
	case marketplace.StateDeleting:
		app.confirmMarketplaceProductDelete(conversation)
	}
}

// Send message to ask for marketplace URL.
func (app *TelegramBotApp) askForMarketplaceUrl(conversation *telegram.Conversation) {
	app.bot.SendMessage(conversation.ChatId, telegram.SendMessageRequest{
		ReplyToMessageId: conversation.LastMessage.MessageId,
		Text:             "Отправь мне ссылку на маркетплейс",
	})

	conversation.StateMachine.TriggerEvent(marketplace.EventWaitForUrl)
}

// Wait for user to enter marketplace URL.
func (app *TelegramBotApp) waitForMarketplaceUrl(conversation *telegram.Conversation) {
	url := conversation.LastMessage.Text
	marketplaceType := marketplace.DetectMarketplaceByUrl(url)

	request := telegram.SendMessageRequest{
		ReplyToMessageId: conversation.LastMessage.MessageId,
	}

	if marketplaceType == marketplace.MarketplaceUnknown {
		request.Text = "Я пока такого маркетплейса не знаю :("

		app.bot.SendMessage(conversation.ChatId, request)
		return
	}

	trackedProduct := conversation.GetContext(telegram.ConversationCtxProduct).(TrackedProduct)

	trackedProduct.marketplace = marketplaceType
	trackedProduct.url = marketplace.GetCleanUrl(url)

	conversation.StoreContext(telegram.ConversationCtxProduct, trackedProduct)

	model, err := app.findUserProductByUrl(conversation.ChatId, conversation.User.Id, trackedProduct.GetUrl())
	if err != nil {
		app.logErrorAndSendMessage(
			conversation,
			err,
			"Unable to find saved product by URL",
			"Не могу найти сохранённый товар по ссылке",
		)
		return
	}

	if model.Exists() {
		request.Text = helpers.ConcatStrings(
			"Уже слежу :)\n\n",
			"Я сообщу, когда цена на товар <b>«<a href=\"", model.Url, "\">", model.Title, "</a>»</b>",
			" станет ниже <b>", helpers.CurrencyFormat(helpers.CurrencyToMajor(model.ThresholdPrice)), "</b>\n",
			"<i>Теущая цена: ", helpers.CurrencyFormat(helpers.CurrencyToMajor(model.CurrentPrice)), "</i>",
		)

		app.bot.SendMessage(conversation.ChatId, request)
		conversation.Reset()
		return
	}

	_, err = conversation.StateMachine.TriggerEvent(marketplace.EventScrape)
	if err != nil {
		app.logErrorAndSendMessage(
			conversation,
			err,
			helpers.ConcatStrings("Unable to trigger state machine \"", string(marketplace.EventScrape), "\" event"),
			"Не могу перейти на следующий шаг",
		)
		return
	}

	// don't wait for next user input, proceed to scraping
	app.scrapeMarketplaceUrl(conversation)
}

// Scrape marketplace URL.
func (app *TelegramBotApp) scrapeMarketplaceUrl(conversation *telegram.Conversation) {
	sentMessage, err := app.bot.SendMessage(conversation.ChatId, telegram.SendMessageRequest{
		ReplyToMessageId: conversation.LastMessage.MessageId,
		Text:             "Ищу...",
	})

	if err != nil {
		app.logger.Println(err)
		return
	}

	request := telegram.EditMessageRequest{}

	loaderDotsCount := 3
	loaderTicker := time.NewTicker(time.Second)
	isLoaderDone := make(chan bool)

	go func() {
		for {
			select {
			case <-isLoaderDone:
				loaderTicker.Stop()

				return
			case <-loaderTicker.C:
				loaderDotsCount++

				if loaderDotsCount > 9 {
					loaderDotsCount = 3
				}

				request.Text = helpers.ConcatStrings("Ищу", strings.Repeat(".", loaderDotsCount))

				app.bot.EditMessage(conversation.ChatId, sentMessage.MessageId, request)
			}
		}
	}()

	trackedProduct := conversation.GetContext(telegram.ConversationCtxProduct).(TrackedProduct)

	scraper := marketplace.NewScraper(app.logger)
	scrapedProduct, err := scraper.Scrape(trackedProduct.GetUrl())

	isLoaderDone <- true

	if err == marketplace.ErrOutOfStock {
		trackedProduct.outOfStock = scrapedProduct.IsOutOfStock()
	} else if err == marketplace.ErrNotFound {
		request.Text = "Не могу найти товар по такой ссылке :("

		app.bot.EditMessage(conversation.ChatId, sentMessage.MessageId, request)
		conversation.Reset()
		return
	} else if err != nil {
		request.Text = "Что-то пошло не так..."

		app.bot.EditMessage(conversation.ChatId, sentMessage.MessageId, request)
		conversation.Reset()
		return
	}

	trackedProduct.scrapedAt = scrapedProduct.GetScrapedAt()
	trackedProduct.title = scrapedProduct.GetTitle()
	trackedProduct.currentPrice = scrapedProduct.GetCurrentPrice()
	trackedProduct.thresholdPrice = trackedProduct.GetCurrentPrice()

	conversation.StoreContext(telegram.ConversationCtxProduct, trackedProduct)

	model, err := app.markerplaceService.Create(&trackedProduct)
	if err != nil {
		app.logger.Println("ERROR! Unable to create product:", err)

		request.Text = "Не могу сохранить найденный товар :("

		app.bot.EditMessage(conversation.ChatId, sentMessage.MessageId, request)
		return
	}

	if model.OutOfStock {
		request.Text = helpers.ConcatStrings(
			"Нет в наличии ", string(telegram.EmojiNeutralFace), "\n\n",
			"Я сообщу, когда товар <b>«<a href=\"", model.Url, "\">", model.Title, "</a>»</b> (", marketplace.GetMarketplaceName(&model), ")",
			" снова поступит в продажу",
		)
	} else {
		request.Text = helpers.ConcatStrings(
			"Окей ", string(telegram.EmojiOkHand), "\n\n",
			"Я сообщу, когда цена на товар <b>«<a href=\"", model.Url, "\">", model.Title, "</a>»</b> (", marketplace.GetMarketplaceName(&model), ")",
			" станет ниже <b>", helpers.CurrencyFormat(helpers.CurrencyToMajor(model.ThresholdPrice)), "</b>",
		)
	}

	app.bot.EditMessage(conversation.ChatId, sentMessage.MessageId, request)
	conversation.Reset()
}

// Show marketplace products listing.
func (app *TelegramBotApp) showMarketplaceListing(conversation *telegram.Conversation) {
	page := 1

	command := conversation.LastMessage.Text

	if strings.HasPrefix(command, telegram.CommandPrefixPage) {
		parts := strings.Split(command, "_")

		requestedPage, err := strconv.Atoi(parts[1])
		if err != nil {
			app.logErrorAndSendMessage(
				conversation,
				err,
				"Unable to parse page number",
				"Не могу определить номер страницы",
			)
			return
		}

		page = requestedPage
	}

	perPage := 5
	result := app.markerplaceService.FindAllPaginated(page, perPage)

	if result.Total == 0 {
		request := telegram.SendMessageRequest{
			Text: helpers.ConcatStrings(
				"У тебя пока нет отслеживаемых товаров\n\n",
				"Воспользуйся командой <code>", telegram.CommandTrackProduct, "</code> чтобы начать",
			),
			ReplyToMessageId: conversation.LastMessage.MessageId,
		}

		app.bot.SendMessage(conversation.ChatId, request)
		conversation.Reset()
		return
	}

	listMessage := "<b>Отслеживаемые товары</b>:\n\n"

	for key, item := range result.Items {
		model := item.(marketplace.Product)

		position := key + 1
		if result.CurrentPage > 1 {
			position += int(result.CurrentPage-1) * int(result.PerPage)
		}

		itemMessage := helpers.ConcatStrings(
			"<b>", strconv.Itoa(position), ". «<a href=\"", model.Url, "\">", model.Title, "</a>»</b>",
			" (", marketplace.GetMarketplaceName(&model), ")",
		)

		if model.OutOfStock {
			itemMessage = helpers.ConcatStrings(
				itemMessage, "\n",
				"<b>Нет в наличии</b>",
				" (", helpers.TimeToHuman(model.ScrapedAt.In(app.timeLocation)), ")",
			)
		} else {
			itemMessage = helpers.ConcatStrings(
				itemMessage, "\n",
				"<b>Текущая цена: ", helpers.CurrencyFormat(helpers.CurrencyToMajor(model.CurrentPrice)), "</b>",
				" (", helpers.TimeToHuman(model.ScrapedAt.In(app.timeLocation)), ")",
			)
		}

		itemMessage = helpers.ConcatStrings(itemMessage, "\n", "<i>Удалить</i>: ", telegram.CommandPrefixDeleteProduct, model.Slug)

		listMessage = helpers.ConcatStrings(listMessage, itemMessage, "\n\n")
	}

	pageNavKeyboard := [][]telegram.InlineKeyboardButton{
		{},
	}

	if pageNav := app.buildPageNavigationKeyboard(result); len(pageNav) > 0 {
		pageNavKeyboard[0] = pageNav
	}

	// edit existing message with products list
	if conversation.LastCallbackQueryId != "" {
		request := telegram.EditMessageRequest{
			Text: listMessage,
		}

		if len(pageNavKeyboard[0]) > 0 {
			request.ReplyMarkup = telegram.InlineKeyboardMarkup{
				Keyboard: pageNavKeyboard,
			}
		}

		sentMessage := conversation.GetContext(telegram.ConversationCtxMessage).(telegram.Message)

		app.bot.EditMessage(conversation.ChatId, sentMessage.MessageId, request)
		app.bot.AnswerCallbackQuery(conversation.LastCallbackQueryId)

		return
	}

	request := telegram.SendMessageRequest{
		Text:             listMessage,
		ReplyToMessageId: conversation.LastMessage.MessageId,
	}

	if len(pageNavKeyboard[0]) > 0 {
		request.ReplyMarkup = telegram.InlineKeyboardMarkup{
			Keyboard: pageNavKeyboard,
		}
	}

	sentMessage, err := app.bot.SendMessage(conversation.ChatId, request)
	if err != nil {
		app.logger.Println("ERROR! Unable to send listing message:", err)
		return
	}

	conversation.StoreContext(telegram.ConversationCtxMessage, sentMessage)
}

// Create page navigation inline keyboard.
func (app *TelegramBotApp) buildPageNavigationKeyboard(result core.PaginatedResult) []telegram.InlineKeyboardButton {
	var keyboard []telegram.InlineKeyboardButton

	if result.Total <= result.PerPage {
		return keyboard
	}

	prevPage := result.CurrentPage - 1
	if prevPage < 1 {
		prevPage = 1
	}

	nextPage := result.CurrentPage + 1
	if nextPage > result.LastPage {
		nextPage = result.LastPage
	}

	if prevPage < result.CurrentPage {
		if prevPage > 1 {
			keyboard = append(keyboard, telegram.InlineKeyboardButton{
				Text:         "« 1",
				CallbackData: helpers.ConcatStrings(telegram.CommandPrefixPage, "1"),
			})
		}

		keyboard = append(keyboard, telegram.InlineKeyboardButton{
			Text:         helpers.ConcatStrings("‹ ", strconv.Itoa(prevPage)),
			CallbackData: helpers.ConcatStrings(telegram.CommandPrefixPage, strconv.Itoa(prevPage)),
		})
	}

	keyboard = append(keyboard, telegram.InlineKeyboardButton{
		Text:         helpers.ConcatStrings("· ", strconv.Itoa(result.CurrentPage), " ·"),
		CallbackData: helpers.ConcatStrings(telegram.CommandPrefixPage, strconv.Itoa(result.CurrentPage)),
	})

	if nextPage > result.CurrentPage {
		keyboard = append(keyboard, telegram.InlineKeyboardButton{
			Text:         helpers.ConcatStrings(strconv.Itoa(nextPage), " ›"),
			CallbackData: helpers.ConcatStrings(telegram.CommandPrefixPage, strconv.Itoa(nextPage)),
		})

		if nextPage < result.LastPage {
			keyboard = append(keyboard, telegram.InlineKeyboardButton{
				Text:         helpers.ConcatStrings(strconv.Itoa(result.LastPage), " »"),
				CallbackData: helpers.ConcatStrings(telegram.CommandPrefixPage, strconv.Itoa(result.LastPage)),
			})
		}
	}

	return keyboard
}

// Confirm marketplace product delete.
func (app *TelegramBotApp) confirmMarketplaceProductDelete(conversation *telegram.Conversation) {
	if conversation.LastMessage.Text == telegram.CommandYes {
		app.bot.AnswerCallbackQuery(conversation.LastCallbackQueryId)
		app.deleteMarketplaceProduct(conversation)
		return
	}

	if conversation.LastMessage.Text == telegram.CommandNo {
		request := telegram.SendMessageRequest{
			Text: helpers.ConcatStrings("Окей ", string(telegram.EmojiOkHand), "\n\nНичего удалять не буду"),
		}

		app.bot.SendMessage(conversation.ChatId, request)
		app.bot.AnswerCallbackQuery(conversation.LastCallbackQueryId)

		conversation.Reset()

		return
	}

	slug := strings.Replace(conversation.LastMessage.Text, telegram.CommandPrefixDeleteProduct, "", 1)
	conversation.StoreContext(telegram.ConversationCtxProductSlug, slug)

	model, err := app.findUserProductBySlug(conversation.ChatId, conversation.LastMessage.From.Id, slug)
	if err != nil {
		app.logErrorAndSendMessage(
			conversation,
			err,
			"Unable to fund user product by slug to confirm",
			"Не удалось найти товар",
		)
		return
	}

	request := telegram.SendMessageRequest{
		ReplyToMessageId: conversation.LastMessage.MessageId,
	}

	if model.Exists() {
		request.Text = helpers.ConcatStrings(
			"Точно хочешь удалить товар",
			" <b>«<a href=\"", model.Url, "\">", model.Title, "</a>»</b>",
			" (", marketplace.GetMarketplaceName(&model), ")", "?",
		)

		request.ReplyMarkup = telegram.InlineKeyboardMarkup{
			Keyboard: [][]telegram.InlineKeyboardButton{
				{
					{
						Text:         "Да",
						CallbackData: telegram.CommandYes,
					},
					{
						Text:         "Нет",
						CallbackData: telegram.CommandNo,
					},
				},
			},
		}
	} else {
		request.Text = "Нет такого товара"
	}

	app.bot.SendMessage(conversation.ChatId, request)
}

// Delete marketplace product.
func (app *TelegramBotApp) deleteMarketplaceProduct(conversation *telegram.Conversation) {
	productSlug := conversation.GetContext(telegram.ConversationCtxProductSlug).(string)

	product, err := app.findUserProductBySlug(conversation.ChatId, conversation.LastMessage.From.Id, productSlug)
	if err != nil {
		app.logErrorAndSendMessage(
			conversation,
			err,
			"Unable to find user product by slug to delete",
			"Не удалось найти товар",
		)
		return
	}

	request := telegram.SendMessageRequest{
		ReplyToMessageId: conversation.LastMessage.MessageId,
	}

	if product.Exists() && app.markerplaceService.Delete(product.Id) {
		request.Text = helpers.ConcatStrings(
			"Товар <b>«<a href=\"", product.Url, "\">", product.Title, "</a>»</b> (", marketplace.GetMarketplaceName(&product), ")",
			" удалён",
		)
	} else {
		request.Text = "Нет такого товара"
	}

	app.bot.SendMessage(conversation.ChatId, request)
	conversation.Reset()
}

// Check if conversation context type is "TrackedProduct".
func (app *TelegramBotApp) isTrackedProductContext(conversation *telegram.Conversation) bool {
	return reflect.TypeOf(conversation.GetContext(telegram.ConversationCtxProduct)) == reflect.TypeOf(TrackedProduct{})
}

// Find user's saved product by URL.
func (app *TelegramBotApp) findUserProductByUrl(telegramChatId int, telegramUserId int, url string) (marketplace.Product, error) {
	model, err := app.markerplaceService.FindForUserByUrl(telegramChatId, telegramUserId, url)
	if err != nil {
		return marketplace.Product{}, err
	}

	return model, nil
}

// Find user's saved product by slug.
func (app *TelegramBotApp) findUserProductBySlug(telegramChatId int, telegramUserId int, slug string) (marketplace.Product, error) {
	model, err := app.markerplaceService.FindForUserBySlug(telegramChatId, telegramUserId, slug)
	if err != nil {
		return marketplace.Product{}, err
	}

	return model, nil
}

// Log error and send message to user.
func (app *TelegramBotApp) logErrorAndSendMessage(conversation *telegram.Conversation, err error, logPrefix string, messageText string) {
	app.logger.Println(helpers.ConcatStrings("ERROR! ", logPrefix, ":"), err)

	request := telegram.SendMessageRequest{
		Text: helpers.ConcatStrings("ОШИБКА! ", messageText),
	}

	app.bot.SendMessage(conversation.ChatId, request)
}
