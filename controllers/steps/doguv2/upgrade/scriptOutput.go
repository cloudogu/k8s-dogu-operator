package upgrade

import (
	"bytes"
	"unicode/utf8"

	"github.com/go-logr/logr"
)

// Leave room for JSON escaping and metadata in container log records.
const scriptOutputChunkBytes = 1024

// scriptOutputWriter logs bounded chunks without waiting for a newline. Each stream has its own writer.
// Only an incomplete UTF-8 character is buffered until the next Write or flush after command execution.
type scriptOutputWriter struct {
	logger  logr.Logger
	pending []byte
}

func (w *scriptOutputWriter) Write(output []byte) (int, error) {
	written := len(output)
	if len(w.pending) > 0 {
		output = append(w.pending, output...)
		w.pending = nil
	}

	for len(output) > 0 {
		end := min(len(output), scriptOutputChunkBytes)
		characterStart := end - 1
		for characterStart > 0 && !utf8.RuneStart(output[characterStart]) {
			characterStart--
		}
		if !utf8.FullRune(output[characterStart:end]) {
			end = characterStart
		}
		if end == 0 {
			w.pending = bytes.Clone(output)
			break
		}

		w.logger.Info("upgrade script output", "output", string(output[:end]))
		output = output[end:]
	}

	return written, nil
}

func (w *scriptOutputWriter) flush() {
	if len(w.pending) > 0 {
		w.logger.Info("upgrade script output", "output", string(w.pending))
		w.pending = nil
	}
}
