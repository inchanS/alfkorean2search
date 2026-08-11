package handlers

import (
	"encoding/json"
	"strings"
	"testing"
)

// decode is a test helper: unmarshal a stdict JSON envelope, exercising the
// flexible item/sense/flexStr unmarshalers, then flatten it.
func decode(t *testing.T, body string) []suggestion {
	t.Helper()
	var res apiResponse
	if err := json.Unmarshal([]byte(body), &res); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	return parseSuggestions(res)
}

// TestParseSuggestionsLove mirrors the original test_workflow.py case: 사랑 with
// homograph number 1 and its definition.
func TestParseSuggestionsLove(t *testing.T) {
	body := `{"channel":{"item":[{"word":"사랑","sup_no":"1","target_code":"12345","link":"",
		"sense":[{"definition":"어떤 사람이나 존재를 몹시 아끼고 귀중히 여기는 마음."}]}]}}`

	got := decode(t, body)
	if len(got) == 0 {
		t.Fatal("expected at least one suggestion")
	}
	if got[0].Word != "사랑(1)" {
		t.Errorf("word = %q, want 사랑(1)", got[0].Word)
	}
	if !strings.Contains(got[0].Definition, "아끼고 귀중히 여기는") {
		t.Errorf("definition = %q", got[0].Definition)
	}
	// link was empty, so it is synthesized from target_code.
	if !strings.Contains(got[0].Link, "word_no=12345") {
		t.Errorf("link should be synthesized from target_code, got %q", got[0].Link)
	}
}

// TestParseSuggestionsSingleObject covers the API collapsing a lone item and a
// lone sense into JSON objects rather than one-element arrays.
func TestParseSuggestionsSingleObject(t *testing.T) {
	body := `{"channel":{"item":{"word":"나무","sup_no":"0","target_code":"999",
		"sense":{"definition":"줄기나 가지가 목질로 된 여러해살이 식물."}}}}`

	got := decode(t, body)
	if len(got) != 1 {
		t.Fatalf("want 1 suggestion, got %d", len(got))
	}
	// sup_no "0" -> no homograph suffix.
	if got[0].Word != "나무" {
		t.Errorf("word = %q, want 나무", got[0].Word)
	}
}

// TestParseSuggestionsStripsMarkup checks that '^' and '-' are removed and an
// explicit link is used verbatim.
func TestParseSuggestionsStripsMarkup(t *testing.T) {
	body := `{"channel":{"item":[{"word":"한-국^말","sup_no":"2","target_code":"7",
		"link":"https://stdict.korean.go.kr/custom","sense":[{"definition":"정의"}]}]}}`

	got := decode(t, body)
	if got[0].Word != "한국말(2)" {
		t.Errorf("word = %q, want 한국말(2)", got[0].Word)
	}
	if got[0].Link != "https://stdict.korean.go.kr/custom" {
		t.Errorf("explicit link should be used, got %q", got[0].Link)
	}
}

// TestParseSuggestionsNumericSupNo checks flexStr tolerates a bare-number
// sup_no (some responses render it unquoted).
func TestParseSuggestionsNumericSupNo(t *testing.T) {
	body := `{"channel":{"item":[{"word":"말","sup_no":3,"target_code":42,
		"sense":[{"definition":"정의"}]}]}}`

	got := decode(t, body)
	if got[0].Word != "말(3)" {
		t.Errorf("word = %q, want 말(3)", got[0].Word)
	}
	if !strings.Contains(got[0].Link, "word_no=42") {
		t.Errorf("link = %q, want word_no=42", got[0].Link)
	}
}

// TestParseSuggestionsMultipleSenses emits one row per sense definition.
func TestParseSuggestionsMultipleSenses(t *testing.T) {
	body := `{"channel":{"item":[{"word":"배","sup_no":"1","target_code":"1",
		"sense":[{"definition":"첫째 뜻"},{"definition":"둘째 뜻"}]}]}}`

	got := decode(t, body)
	if len(got) != 2 {
		t.Fatalf("want 2 rows, got %d", len(got))
	}
	if got[0].Definition != "첫째 뜻" || got[1].Definition != "둘째 뜻" {
		t.Errorf("definitions = %q, %q", got[0].Definition, got[1].Definition)
	}
}

// TestParseSuggestionsEmpty returns nothing for an envelope with no items.
func TestParseSuggestionsEmpty(t *testing.T) {
	if got := decode(t, `{"channel":{}}`); got != nil {
		t.Errorf("want nil, got %#v", got)
	}
}
