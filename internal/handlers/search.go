package handlers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/inchanS/AlfKorean2Search/internal/alfred"
	"github.com/inchanS/AlfKorean2Search/internal/cache"
	"github.com/inchanS/AlfKorean2Search/internal/httpx"
)

// stdict is 국립국어원 표준국어대사전 (stdict.korean.go.kr). Its OpenAPI
// search endpoint returns a JSON envelope of the form
//
//	{"channel": {"item": [{"word", "sup_no", "target_code", "link",
//	                        "sense": [{"definition"}]}]}}
//
// where "item" and "sense" collapse to a single object (not an array) when
// there is exactly one entry — handled by the flexible unmarshalers below.
const (
	apiURL           = "https://stdict.korean.go.kr/api/search.do"
	generalSearchURL = "https://stdict.korean.go.kr/search/searchResult.do?pageSize=10&searchKeyword=%s"
	// viewURL is synthesized as a detail link when the API omits "link".
	viewURL        = "https://stdict.korean.go.kr/search/searchView.do?word_no=%s&searchKeywordTo=3"
	searchCacheAge = time.Hour

	// cachePrefix keys every per-query entry; pruneStampKey (a distinct, fixed
	// key not sharing that prefix) gates how often stale entries are swept.
	cachePrefix   = "stdict"
	pruneStampKey = "__stdict_prune"
)

// suggestion is one display row: a word (with homograph number), its
// definition, and the detail page URL.
type suggestion struct {
	Word       string
	Definition string
	Link       string
}

// search is the 표준국어대사전 lookup handler, ported from korean_search.py.
func search(fb *alfred.Feedback, word string) error {
	if word == "" {
		return nil
	}

	apiKey := os.Getenv("API_KEY")
	if apiKey == "" {
		fb.Add(alfred.ItemOpts{
			Title:    "API 키가 설정되지 않았습니다.",
			Subtitle: "Alfred 워크플로우의 [Environment Variables]에서 API_KEY를 입력해주세요.",
			Icon:     iconNoResults,
			Valid:    false,
		})
		return nil
	}

	// Head item: open the full search result page for the raw query.
	fb.Add(alfred.ItemOpts{
		Title:        "'" + word + "' 전체 검색하기",
		Subtitle:     "표준국어대사전에서 '" + word + "'의 전체 검색 결과를 확인합니다.",
		Autocomplete: word,
		Arg:          quick(generalSearchURL, word),
		QuicklookURL: quick(generalSearchURL, word),
		Valid:        true,
	})

	maybePruneCache()

	body, err := cache.Cached(cache.Key(cachePrefix, word), searchCacheAge, func() ([]byte, error) {
		return httpx.Get(apiURL, map[string]string{
			"key":      apiKey,
			"q":        word,
			"req_type": "json",
		}, nil)
	})
	if err != nil {
		// Keep the head item and append the error below it, as korean_search.py
		// did — the user can still run the full search.
		addAPIError(fb, err)
		return nil
	}
	if body == nil {
		return nil
	}

	var res apiResponse
	if err := json.Unmarshal(body, &res); err != nil {
		addAPIError(fb, err)
		return nil
	}

	suggestions := parseSuggestions(res)

	// Deduplicate by word+definition, preserving order (mirrors korean_search.py).
	seen := map[string]bool{}
	for _, s := range suggestions {
		key := s.Word + "_" + s.Definition
		if seen[key] {
			continue
		}
		seen[key] = true

		fb.Add(alfred.ItemOpts{
			Title:        s.Word,
			Subtitle:     s.Definition,
			Autocomplete: s.Word,
			Arg:          s.Link,
			Copy:         s.Word,
			LargeType:    s.Definition,
			QuicklookURL: s.Link,
			Valid:        true,
		})
	}
	return nil
}

// maybePruneCache sweeps stale per-query cache entries at most once per
// searchCacheAge. Alfred spawns this binary on every keystroke, so the sweep is
// gated by a stamp file to avoid a directory scan on each run: only the first
// search after the stamp goes stale performs the (cheap) prune. Without it, each
// distinct query would leave a stdict_<md5>.json behind forever.
func maybePruneCache() {
	if _, fresh := cache.Read(pruneStampKey, searchCacheAge); fresh {
		return
	}
	_ = cache.Write(pruneStampKey, []byte(time.Now().Format(time.RFC3339)))
	_ = cache.Prune(cachePrefix, searchCacheAge)
}

// addAPIError appends the standard API-error row (matching korean_search.py's
// message) without clearing items already added.
func addAPIError(fb *alfred.Feedback, err error) {
	fb.Add(alfred.ItemOpts{
		Title:    "API 요청 중 오류가 발생했습니다. 사전에 없는 검색어일 수 있습니다.",
		Subtitle: err.Error(),
		Icon:     iconNoResults,
		Valid:    false,
	})
}

// apiResponse is the decoded stdict OpenAPI JSON envelope.
type apiResponse struct {
	Channel struct {
		Item items `json:"item"`
	} `json:"channel"`
}

type item struct {
	Word       string  `json:"word"`
	SupNo      flexStr `json:"sup_no"`
	TargetCode flexStr `json:"target_code"`
	Link       string  `json:"link"`
	Sense      senses  `json:"sense"`
}

type sense struct {
	Definition string `json:"definition"`
}

// parseSuggestions flattens the API response into display rows, mirroring
// parse_suggestions() in korean_search.py: strip API markup ('^', '-') from
// the word, append the homograph number (사랑(1)) when present, synthesize a
// detail link from target_code when "link" is absent, and emit one row per
// sense definition. Duplicates are left in place; the caller deduplicates.
func parseSuggestions(res apiResponse) []suggestion {
	stripper := strings.NewReplacer("^", "", "-", "")

	var out []suggestion
	for _, it := range res.Channel.Item {
		pureWord := stripper.Replace(it.Word)

		display := pureWord
		if s := string(it.SupNo); s != "" && s != "0" {
			display = fmt.Sprintf("%s(%s)", pureWord, s)
		}

		link := it.Link
		if link == "" && it.TargetCode != "" {
			link = fmt.Sprintf(viewURL, string(it.TargetCode))
		}

		for _, sn := range it.Sense {
			def := sn.Definition
			if def == "" {
				def = "설명 없음"
			}
			out = append(out, suggestion{Word: display, Definition: def, Link: link})
		}
	}
	return out
}

// The stdict API renders a lone item/sense as a JSON object rather than a
// one-element array. items and senses accept both shapes.

type items []item

func (it *items) UnmarshalJSON(data []byte) error {
	return unmarshalSingleOrArray(data, (*[]item)(it))
}

type senses []sense

func (s *senses) UnmarshalJSON(data []byte) error {
	return unmarshalSingleOrArray(data, (*[]sense)(s))
}

// unmarshalSingleOrArray decodes data into *out, accepting either a JSON array
// of T or a single T object (which becomes a one-element slice).
func unmarshalSingleOrArray[T any](data []byte, out *[]T) error {
	trimmed := bytes.TrimSpace(data)
	if len(trimmed) == 0 || string(trimmed) == "null" {
		return nil
	}
	if trimmed[0] == '[' {
		return json.Unmarshal(trimmed, out)
	}
	var single T
	if err := json.Unmarshal(trimmed, &single); err != nil {
		return err
	}
	*out = append(*out, single)
	return nil
}

// flexStr decodes a JSON value that stdict may render as either a string or a
// bare number (e.g. sup_no, target_code) into a string.
type flexStr string

func (f *flexStr) UnmarshalJSON(data []byte) error {
	data = bytes.TrimSpace(data)
	if len(data) == 0 || string(data) == "null" {
		*f = ""
		return nil
	}
	if data[0] == '"' {
		var s string
		if err := json.Unmarshal(data, &s); err != nil {
			return err
		}
		*f = flexStr(s)
		return nil
	}
	*f = flexStr(data) // bare number, kept as-is
	return nil
}
