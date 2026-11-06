package output

import (
	"encoding/json"
	"io"

	"github.com/seletz/odoo-work-cli/internal/apperr"
)

// encoder returns a JSON encoder for f: indented for json, compact for
// jsonl. HTML escaping is off so names like "R&D" stay readable.
func encoder(w io.Writer, f Format) *json.Encoder {
	enc := json.NewEncoder(w)
	enc.SetEscapeHTML(false)
	if f == FormatJSON {
		enc.SetIndent("", "  ")
	}
	return enc
}

// WriteList writes items as a JSON array (json) or one object per line
// (jsonl). An empty list is always written as "[]", in both formats, so
// that an empty result is never empty output.
//
// :param w: destination
// :param f: FormatJSON or FormatJSONL
// :param items: the records
// :returns: a write error
func WriteList[T any](w io.Writer, f Format, items []T) error {
	if items == nil {
		items = []T{}
	}
	enc := encoder(w, f)
	if f == FormatJSONL && len(items) > 0 {
		for _, it := range items {
			if err := enc.Encode(it); err != nil {
				return err
			}
		}
		return nil
	}
	return enc.Encode(items)
}

// WriteObject writes a single object.
//
// :param w: destination
// :param f: FormatJSON or FormatJSONL
// :param v: the object
// :returns: a write error
func WriteObject(w io.Writer, f Format, v any) error {
	return encoder(w, f).Encode(v)
}

// ErrorBody is the payload of the JSON error envelope.
type ErrorBody struct {
	// Kind is the stable error kind (see apperr).
	Kind apperr.Kind `json:"kind"`
	// Message is the human-readable message; not stable.
	Message string `json:"message"`
}

// ErrorEnvelope is the single JSON object printed on failure:
// {"error":{"kind":…,"message":…}}.
type ErrorEnvelope struct {
	Error ErrorBody `json:"error"`
}

// WriteError writes err as an ErrorEnvelope. Write errors are ignored:
// there is nowhere left to report them.
//
// :param w: destination (stdout)
// :param f: FormatJSON or FormatJSONL
// :param err: the error to render
func WriteError(w io.Writer, f Format, err error) {
	_ = encoder(w, f).Encode(ErrorEnvelope{Error: ErrorBody{
		Kind:    apperr.KindOf(err),
		Message: err.Error(),
	}})
}
