// Package alfred builds Alfred Script Filter JSON feedback, replacing the
// wf.add_item / send_feedback API previously provided by alfred-pyworkflow.
package alfred

import (
	"encoding/json"
	"os"
)

// Text maps to an item's copy / large-type text.
type Text struct {
	Copy      string `json:"copy,omitempty"`
	LargeType string `json:"largetype,omitempty"`
}

// Icon maps to an item's icon path.
type Icon struct {
	Path string `json:"path,omitempty"`
}

// Item is a single Alfred result row.
type Item struct {
	Title        string         `json:"title"`
	Subtitle     string         `json:"subtitle,omitempty"`
	Arg          string         `json:"arg,omitempty"`
	Autocomplete string         `json:"autocomplete,omitempty"`
	Valid        bool           `json:"valid"`
	QuicklookURL string         `json:"quicklookurl,omitempty"`
	Icon         *Icon          `json:"icon,omitempty"`
	Text         *Text          `json:"text,omitempty"`
	Variables    map[string]any `json:"variables,omitempty"`
}

// Feedback is the top-level Script Filter response.
type Feedback struct {
	Items []*Item `json:"items"`
}

// New returns an empty feedback with a non-nil (JSON "[]") item list.
func New() *Feedback {
	return &Feedback{Items: []*Item{}}
}

// ItemOpts collects the fields callers set, mirroring the previous
// wf.add_item keyword arguments.
type ItemOpts struct {
	Title        string
	Subtitle     string
	Arg          string
	Autocomplete string
	QuicklookURL string
	Copy         string // -> text.copy   (was copytext)
	LargeType    string // -> text.largetype (was largetext)
	Icon         string // icon file path; empty uses the workflow default
	Valid        bool
}

// Add appends an item built from opts and returns it for further tweaks
// (e.g. SetVar).
func (f *Feedback) Add(o ItemOpts) *Item {
	it := &Item{
		Title:        o.Title,
		Subtitle:     o.Subtitle,
		Arg:          o.Arg,
		Autocomplete: o.Autocomplete,
		Valid:        o.Valid,
		QuicklookURL: o.QuicklookURL,
	}
	if o.Copy != "" || o.LargeType != "" {
		it.Text = &Text{Copy: o.Copy, LargeType: o.LargeType}
	}
	if o.Icon != "" {
		it.Icon = &Icon{Path: o.Icon}
	}
	f.Items = append(f.Items, it)
	return it
}

// Reset drops all accumulated items (used to show only an error row).
func (f *Feedback) Reset() {
	f.Items = f.Items[:0]
}

// SetVar attaches an Alfred workflow variable to the item. The value type is
// preserved in JSON (string, bool, …) to match the previous library output that
// info.plist conditionals were written against.
func (it *Item) SetVar(k string, v any) *Item {
	if it.Variables == nil {
		it.Variables = map[string]any{}
	}
	it.Variables[k] = v
	return it
}

// Send writes the feedback as JSON to stdout for Alfred to consume.
func (f *Feedback) Send() {
	// stdout is reserved for Alfred feedback; encoding errors are non-recoverable
	// here and would only corrupt output, so they are intentionally ignored.
	_ = json.NewEncoder(os.Stdout).Encode(f)
}
