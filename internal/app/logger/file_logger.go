package logger

import (
	"bot/internal/app/helpers"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type LoggerInterface interface {
	Println(message ...any)
}

type FileLogger struct {
	path         string
	isSilent     bool
	timeLocation *time.Location
}

func NewFileLogger(fileName string, isSilent bool) FileLogger {
	rootDir, err := helpers.GetRootDir()
	if err != nil {
		fmt.Println("Unable to initialize logger:", err)
	}

	if !strings.HasSuffix(fileName, ".log") {
		helpers.ConcatStrings(fileName, ".log")
	}

	timezone := os.Getenv("TIMEZONE")
	timeLocation, _ := time.LoadLocation(timezone)

	return FileLogger{
		path:         filepath.Join(rootDir, "logs", fileName),
		isSilent:     isSilent,
		timeLocation: timeLocation,
	}
}

func (l FileLogger) Println(message ...any) {
	logFile, err := os.OpenFile(l.path, os.O_APPEND|os.O_RDWR|os.O_CREATE, 0644)
	if err != nil {
		log.Panic(err)
	}

	defer logFile.Close()

	log.SetOutput(logFile)
	log.SetFlags(0)

	timestampPrefix := helpers.ConcatStrings("[", helpers.TimeToDatabase(time.Now().In(l.timeLocation)), "]: ")
	log.SetPrefix(timestampPrefix)

	if !l.isSilent {
		fmt.Println(message...)
	}

	log.Println(message...)
}
