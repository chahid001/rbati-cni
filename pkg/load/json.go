package load

import (
	"io"
	"os"
)

func RawJson()[]byte {

	data, err := io.ReadAll(os.Stdin)

	if err != nil {
		os.Stderr.WriteString("Failed to read stdin: " + err.Error() + "\n")
		os.Exit(1)
	}

	return data
}