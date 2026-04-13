package output

import (
	"encoding/json"
	"fmt"
	"io"

	"github.com/justincampbell/gh-watch/internal/events"
)

// Writer handles formatting events to stdout (JSON) and stderr (human-friendly).
type Writer struct {
	JSON   io.Writer
	Stderr io.Writer
	Quiet  bool // suppress stderr output
}

func (w *Writer) WriteEvents(evts []events.Event) {
	for _, e := range evts {
		w.writeJSON(e)
		if !w.Quiet {
			w.writeHuman(e)
		}
	}
}

func (w *Writer) writeJSON(e events.Event) {
	data, err := json.Marshal(e)
	if err != nil {
		return
	}
	fmt.Fprintln(w.JSON, string(data))
}

func (w *Writer) writeHuman(e events.Event) {
	fmt.Fprintf(w.Stderr, "[%s] %s\n", e.Timestamp.Format("15:04:05"), e.Summary)
}
