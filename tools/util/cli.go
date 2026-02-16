package util

import (
	"fmt"
	"os"
	"time"
)

func Spinner(text string, stop chan bool) {
	// A common set of spinner characters
	frames := []string{"-", "\\", "|", "/"}
	for {
		select {
		case <-stop:
			return
		default:
			for _, frame := range frames {
				// Use \r to return the cursor to the beginning of the line
				// and overwrite the previous frame.
				// The extra spaces are to clear any remaining characters from the previous line.
				fmt.Printf("\r%s %s... ", frame, text)
				// Flush the output buffer
				os.Stdout.Sync()
				time.Sleep(100 * time.Millisecond) // Control the spinner speed
			}
		}
	}
}
