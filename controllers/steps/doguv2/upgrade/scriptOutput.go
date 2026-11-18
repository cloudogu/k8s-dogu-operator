package upgrade

import (
	"bytes"
	"unicode/utf8"

	"github.com/go-logr/logr"
)

// Leave room for JSON escaping and metadata in container log records.
const scriptOutputChunkBytes = 1024

// scriptOutputWriter logs bounded chunks without waiting for a newline. Each stream has its own writer.
// Trailing CR/LF are omitted from lines and final fragments; other bytes are retained.
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
		lineEnd := len(output)
		newline := bytes.IndexByte(output, '\n')
		if newline >= 0 {
			lineEnd = newline + 1
		}

		// Trim before chunking so internal carriage returns at chunk boundaries survive.
		line := bytes.TrimRight(output[:lineEnd], "\r\n")
		lineTerminated := len(line) < lineEnd
		for len(line) > 0 {
			end := min(len(line), scriptOutputChunkBytes)
			characterStart := end - 1
			for characterStart > 0 && !utf8.RuneStart(line[characterStart]) {
				characterStart--
			}
			// A line ending terminates an incomplete UTF-8 sequence; it must not join the next line.
			if !utf8.FullRune(line[characterStart:end]) && (end < len(line) || !lineTerminated) {
				end = characterStart
			}
			if end == 0 {
				w.pending = bytes.Clone(line)
				break
			}

			w.logger.Info("upgrade script output", "output", string(line[:end]))
			line = line[end:]
		}
		output = output[lineEnd:]
	}

	return written, nil
}

func (w *scriptOutputWriter) flush() {
	if len(w.pending) > 0 {
		w.logger.Info("upgrade script output", "output", string(w.pending))
		w.pending = nil
	}
}
