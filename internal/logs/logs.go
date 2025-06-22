package logs

import (
	"log/slog"
	"os"
)

var Log *slog.Logger

func Init(path string) {
	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		panic("failed to open log file: " + err.Error())
	}

	handler := slog.NewJSONHandler(f, nil)
	Log = slog.New(handler)
}
