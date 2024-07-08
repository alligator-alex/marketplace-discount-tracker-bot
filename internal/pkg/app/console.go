package app

import (
	"bot/internal/app/database"
	"fmt"
	"os"
	"strings"
)

const ConsoleAppKeyword = "impulse101"

type ConsoleApp struct {
	db *database.Postgres
}

func NewConsoleApp() (ConsoleApp, error) {
	return ConsoleApp{}, nil
}

func (app ConsoleApp) Run() error {
	fmt.Println("starting console...")

	var err error
	app.db, err = database.NewPostgres()

	defer app.db.CloseConnection()

	if err != nil {
		return err
	}

	app.listenToCommands()

	return nil
}

func (app ConsoleApp) listenToCommands() {
	args := os.Args[1:]

	if len(args) < 2 {
		fmt.Println("Not enough arguments")
		return
	}

	if args[0] != ConsoleAppKeyword {
		fmt.Println("Invalid keyword")
		return
	}

	command := args[1]

	switch command {
	case "migrate":
		app.db.Migrate()
	case "migrate:rollback":
		app.db.RollbackMigrations()
	case "create:migration":
		if (len(args) < 3) || (strings.TrimSpace(args[2]) == "") {
			fmt.Printf("No migration name given")
			return
		}

		database.CreateNewMigration(strings.TrimSpace(args[2]))
	default:
		fmt.Printf("Unknown command \"%s\"\n", command)
	}
}
