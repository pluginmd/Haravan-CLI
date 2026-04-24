package cmdutil

import (
	"encoding/json"
	"io"
)

// WriteJSON encodes v as indented JSON followed by a newline.
func WriteJSON(w io.Writer, v any) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(v)
}
