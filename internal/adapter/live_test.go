package adapter

import (
	"context"
	"testing"
	"time"

	"github.com/andrey-vk/wdsplit/internal/model"
)

// TestLiveV2flyYoutube exercises the full production path — Run(), the
// real SSRF-hardened client, a real public host — against
// v2fly/domain-list-community's live youtube list, replicating in JS the
// same parsing logic the retired compiled-Go v2fly adapter had (full:/
// domain:/keyword:/bare-domain, @attribute and inline-comment stripping,
// regexp:/include: skipped). Needs network access, so it's excluded from
// `go test -short` (what CI runs) and only runs on an explicit `go test
// -run TestLive ./internal/adapter/...`.
func TestLiveV2flyYoutube(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping live network test in -short mode")
	}

	source := `
function sync(feed, api) {
    var text = api.httpGet(feed.url);
    var lines = text.split("\n");
    var filters = [];
    for (var i = 0; i < lines.length; i++) {
        var line = lines[i];

        var hashIdx = line.indexOf("#");
        if (hashIdx >= 0) line = line.substring(0, hashIdx);
        var atIdx = line.indexOf("@");
        if (atIdx >= 0) line = line.substring(0, atIdx);
        line = line.trim();
        if (line === "") continue;

        if (line.indexOf("include:") === 0) continue;
        if (line.indexOf("regexp:") === 0) continue;

        if (line.indexOf("full:") === 0) {
            filters.push({type: "domain", value: line.substring(5)});
        } else if (line.indexOf("domain:") === 0) {
            filters.push({type: "domain_suffix", value: line.substring(7)});
        } else if (line.indexOf("keyword:") === 0) {
            filters.push({type: "domain_keyword", value: line.substring(8)});
        } else {
            filters.push({type: "domain_suffix", value: line});
        }
    }
    return [{category: "Streaming", service: "YouTube", filters: filters}];
}
`
	limits := Limits{
		Timeout:          15 * time.Second,
		MaxSourceBytes:   50_000,
		MaxResponseBytes: 200_000,
		MaxTotalBytes:    200_000,
		MaxRequests:      3,
		MaxEntries:       5000,
		MaxCallStack:     1000,
	}
	feed := Feed{
		ID:   1,
		Name: "v2fly-youtube-live-test",
		URL:  "https://raw.githubusercontent.com/v2fly/domain-list-community/master/data/youtube",
	}

	groups, err := Run(context.Background(), source, feed, limits)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}

	if len(groups) != 1 {
		t.Fatalf("got %d groups, want 1", len(groups))
	}
	g := groups[0]
	if g.Category != "Streaming" || g.Service != "YouTube" {
		t.Errorf("group = %s/%s, want Streaming/YouTube", g.Category, g.Service)
	}
	if len(g.Entries) < 100 {
		t.Errorf("got %d entries, want at least 100 (sanity check against the real list shrinking dramatically)", len(g.Entries))
	}

	want := map[string]model.EntryType{
		"youtube":              model.EntryDomainSuffix, // bare single-label line in the real file
		"googlevideo.com":      model.EntryDomainSuffix,
		"youtube-nocookie.com": model.EntryDomainSuffix,
	}
	found := map[string]model.EntryType{}
	for _, e := range g.Entries {
		if _, ok := want[e.Value]; ok {
			found[e.Value] = e.Type
		}
	}
	for value, wantType := range want {
		gotType, ok := found[value]
		if !ok {
			t.Errorf("expected entry %q not found in live result", value)
			continue
		}
		if gotType != wantType {
			t.Errorf("entry %q type = %s, want %s", value, gotType, wantType)
		}
	}
}
