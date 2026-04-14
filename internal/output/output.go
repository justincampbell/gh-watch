package output

import (
	"encoding/json"
	"fmt"
	"io"

	"github.com/justincampbell/gh-watch/internal/events"
)

// Writer writes JSON events to stdout.
type Writer struct {
	JSON io.Writer
}

func (w *Writer) WriteEvents(evts []events.Event) {
	for _, e := range evts {
		data, err := json.Marshal(e)
		if err != nil {
			return
		}
		fmt.Fprintln(w.JSON, string(data))
	}
}
