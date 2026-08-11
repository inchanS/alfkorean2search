package alfred

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestItemJSON(t *testing.T) {
	fb := New()
	it := fb.Add(ItemOpts{
		Title:        "t",
		Arg:          "a",
		Autocomplete: "ac",
		Copy:         "c",
		LargeType:    "l",
		Icon:         "icon.png",
		Valid:        true,
	})
	it.SetVar("lang", "koko")
	it.SetVar("flag", false) // must serialize as JSON bool, not "false"

	data, err := json.Marshal(fb)
	if err != nil {
		t.Fatal(err)
	}
	var got map[string]any
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatal(err)
	}
	item := got["items"].([]any)[0].(map[string]any)

	if item["text"].(map[string]any)["copy"] != "c" {
		t.Error("copytext should map to text.copy")
	}
	if item["icon"].(map[string]any)["path"] != "icon.png" {
		t.Error("icon should map to icon.path")
	}
	vars := item["variables"].(map[string]any)
	if vars["lang"] != "koko" {
		t.Error("lang variable missing")
	}
	// A bool variable must remain a JSON bool so info.plist conditionals match
	// the previous library's output.
	if v, ok := vars["flag"].(bool); !ok || v != false {
		t.Errorf("flag should be JSON bool false, got %#v", vars["flag"])
	}
}

func TestValidAlwaysEmitted(t *testing.T) {
	fb := New()
	fb.Add(ItemOpts{Title: "x", Valid: false})
	data, _ := json.Marshal(fb)
	// "valid":false must be present (not omitted) so Alfred does not default it.
	if !json.Valid(data) || !strings.Contains(string(data), `"valid":false`) {
		t.Errorf("expected explicit valid:false in %s", data)
	}
}

func TestResetClearsItems(t *testing.T) {
	fb := New()
	fb.Add(ItemOpts{Title: "a"})
	fb.Add(ItemOpts{Title: "b"})
	fb.Reset()
	if len(fb.Items) != 0 {
		t.Fatalf("Reset should clear items, got %d", len(fb.Items))
	}
	// After Reset the list must still marshal as "[]", not null.
	data, _ := json.Marshal(fb)
	if !strings.Contains(string(data), `"items":[]`) {
		t.Errorf("expected empty items array, got %s", data)
	}
}
