package main

import (
	"bot/internal/app/helpers"
	"bot/internal/app/logger"
	"bot/internal/pkg/app"
	"log"
	"os"
	"path/filepath"
	"strconv"

	"github.com/joho/godotenv"
)

func main() {
	if err := loadEnv(); err != nil {
		log.Fatalln("Unable to load .env file:", err)
	}

	if checkIsCliMode() {
		runConsoleApp()
		return
	}

	runBotApp()
}

func loadEnv() error {
	rootDir, err := helpers.GetRootDir()

	if err != nil {
		return err
	}

	filePath := filepath.Join(rootDir, ".env")

	return godotenv.Load(filePath)
}

func checkIsCliMode() bool {
	args := os.Args[1:]

	return len(args) > 1 && args[0] == app.ConsoleAppKeyword
}

func runConsoleApp() {
	app, err := app.NewConsoleApp()
	if err != nil {
		log.Fatal(err)
	}

	err = app.Run()
	if err != nil {
		log.Fatal(err)
	}
}

func runBotApp() {
	token := os.Getenv("TELEGRAM_BOT_TOKEN")

	logger := logger.NewFileLogger("app.log", false)

	scraperTimeoutInSeconds, err := strconv.Atoi(os.Getenv("SCRAPER_TIMEOUT_IN_SECONDS"))
	if err != nil {
		scraperTimeoutInSeconds = 60
	}

	watcherIntervalInMinutes, err := strconv.Atoi(os.Getenv("WATCHER_INTERVAL_IN_MINUTES"))
	if err != nil {
		watcherIntervalInMinutes = 60
	}

	app := app.NewTelegramBotApp(token, logger, scraperTimeoutInSeconds, watcherIntervalInMinutes)
	app.Run()
}
