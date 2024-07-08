package telegram

import (
	"bot/internal/app/statemachine"
	"context"
)

type ConversationContextKey string

const (
	ConversationCtxProduct     ConversationContextKey = "trackedProduct"
	ConversationCtxMessage     ConversationContextKey = "message"
	ConversationCtxProductSlug ConversationContextKey = "productSlug"
)

type Conversation struct {
	ChatId              int
	User                User
	LastMessage         Message
	LastCallbackQueryId string
	StateMachine        statemachine.StateMachine
	ctx                 context.Context
}

func NewConversation(chatId int, from User) *Conversation {
	return &Conversation{
		ChatId: chatId,
		User:   from,
		ctx:    context.Background(),
	}
}

// Get conversation context by key.
func (c *Conversation) GetContext(key ConversationContextKey) any {
	if c.ctx == nil {
		return nil
	}

	return c.ctx.Value(key)
}

// Store conversation context by key.
func (c *Conversation) StoreContext(key ConversationContextKey, value any) any {
	if c.ctx == nil {
		c.ctx = context.Background()
	}

	c.ctx = context.WithValue(c.ctx, key, value)

	return c.ctx
}

// Reset conversation.
func (c *Conversation) Reset() {
	c.ctx = nil
	c.StateMachine.Reset()
}
