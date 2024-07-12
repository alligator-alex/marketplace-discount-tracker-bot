package marketplace

import "bot/internal/app/statemachine"

const (
	StateAskingForUrl  statemachine.State = "AskingForUrl"
	StateWaitingForUrl statemachine.State = "WaitingForUrl"
	StateScraping      statemachine.State = "Scraping"
	StateListing       statemachine.State = "Listing"
	StateDeleting      statemachine.State = "Deleting"
)

const (
	EventAskForUrl  statemachine.Event = "AskForUrl"
	EventWaitForUrl statemachine.Event = "WaitForUrl"
	EventScrape     statemachine.Event = "Scrape"
	EventList       statemachine.Event = "List"
	EventDelete     statemachine.Event = "Delete"
)

func NewFsm() statemachine.StateMachine {
	transitions := statemachine.TransitionsList{
		EventAskForUrl: {
			From: []statemachine.State{
				statemachine.StateIdle,
				StateDeleting,
			},
			To: StateAskingForUrl,
		},

		EventWaitForUrl: {
			From: []statemachine.State{
				StateAskingForUrl,
			},
			To: StateWaitingForUrl,
		},

		EventScrape: {
			From: []statemachine.State{
				StateWaitingForUrl,
			},
			To: StateScraping,
		},

		EventList: {
			From: []statemachine.State{
				statemachine.StateIdle,
				StateDeleting,
			},
			To: StateListing,
		},

		EventDelete: {
			From: []statemachine.State{
				statemachine.StateIdle,
				StateListing,
			},
			To: StateDeleting,
		},
	}

	return statemachine.NewFSM(statemachine.StateIdle, transitions)
}
