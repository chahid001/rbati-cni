package logs

import (
	"log/slog"
	"os"
	"path/filepath"
	"rbati-cni/internal/types"
)

var Log *slog.Logger

func Init() {

	if err := os.MkdirAll(types.LOG_FILE_PATH, 0755); err != nil {
		panic("failed to MKDIR: " + err.Error())
	}

	path := filepath.Join(types.LOG_FILE_PATH, types.LOG_FILE_FILENAME)

	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		panic("failed to open log file: " + err.Error())
	}

	handler := slog.NewJSONHandler(f, nil)
	Log = slog.New(handler)
}
