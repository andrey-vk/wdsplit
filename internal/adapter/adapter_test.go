package adapter

import (
	"context"
	"strings"
	"testing"
	"time"
)

func testLimits() Limits {
	return Limits{
		Timeout:          2 * time.Second,
		MaxSourceBytes:   10_000,
		MaxResponseBytes: 10_000,
		MaxTotalBytes:    10_000,
		MaxRequests:      5,
		MaxEntries:       100,
		MaxCallStack:     500,
	}
}

func TestRunHappyPath(t *testing.T) {
	source := `
function sync(feed, api) {
    return [
        {category: "Streaming", service: "Example", filters: [
            {type: "domain_suffix", value: "example.com"},
            {type: "domain", value: "exact.example.com"},
            {type: "cidr", value: "203.0.113.0/24"}
        ]}
    ];
}
`
	groups, err := Run(context.Background(), source, Feed{ID: 1, Name: "test", URL: "https://example.invalid/list"}, testLimits())
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if len(groups) != 1 {
		t.Fatalf("got %d groups, want 1", len(groups))
	}
	g := groups[0]
	if g.Category != "Streaming" || g.Service != "Example" {
		t.Errorf("group = %+v, want Category=Streaming Service=Example", g)
	}
	if len(g.Entries) != 3 {
		t.Fatalf("got %d entries, want 3: %+v", len(g.Entries), g.Entries)
	}
}

func TestRunReceivesFeedFields(t *testing.T) {
	source := `
function sync(feed, api) {
    if (feed.id !== 42) throw new Error("bad id: " + feed.id);
    if (feed.name !== "my feed") throw new Error("bad name: " + feed.name);
    if (feed.url !== "https://example.invalid/x") throw new Error("bad url: " + feed.url);
    var cfg = JSON.parse(feed.data);
    if (cfg.foo !== "bar") throw new Error("bad data: " + feed.data);
    return [];
}
`
	_, err := Run(context.Background(), source, Feed{
		ID: 42, Name: "my feed", URL: "https://example.invalid/x", Data: `{"foo":"bar"}`,
	}, testLimits())
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
}

func TestRunRequiresSyncFunction(t *testing.T) {
	_, err := Run(context.Background(), `var x = 1;`, Feed{}, testLimits())
	if err == nil {
		t.Fatal("Run() error = nil, want error (no sync function defined)")
	}
	if !strings.Contains(err.Error(), "sync") {
		t.Errorf("error = %v, want it to mention 'sync'", err)
	}
}

func TestRunScriptThrowIsReturnedAsError(t *testing.T) {
	source := `function sync(feed, api) { throw new Error("deliberate failure"); }`
	_, err := Run(context.Background(), source, Feed{}, testLimits())
	if err == nil {
		t.Fatal("Run() error = nil, want error")
	}
	if !strings.Contains(err.Error(), "deliberate failure") {
		t.Errorf("error = %v, want it to contain the thrown message", err)
	}
}

func TestRunSourceTooLarge(t *testing.T) {
	limits := testLimits()
	limits.MaxSourceBytes = 10
	_, err := Run(context.Background(), `function sync(feed, api) { return []; }`, Feed{}, limits)
	if err == nil {
		t.Fatal("Run() error = nil, want error (source exceeds max bytes)")
	}
}

func TestRunTimeout(t *testing.T) {
	limits := testLimits()
	limits.Timeout = 100 * time.Millisecond
	source := `function sync(feed, api) { while (true) {} }`

	start := time.Now()
	_, err := Run(context.Background(), source, Feed{}, limits)
	elapsed := time.Since(start)

	if err == nil {
		t.Fatal("Run() error = nil, want timeout error")
	}
	if elapsed > 2*time.Second {
		t.Errorf("Run() took %v, want it to be interrupted near the 100ms timeout", elapsed)
	}
}

func TestRunMaxCallStack(t *testing.T) {
	limits := testLimits()
	limits.MaxCallStack = 20
	source := `
function recurse(n) { return recurse(n + 1); }
function sync(feed, api) { return recurse(0); }
`
	_, err := Run(context.Background(), source, Feed{}, limits)
	if err == nil {
		t.Fatal("Run() error = nil, want stack overflow error")
	}
}

func TestRunInvalidReturnShape(t *testing.T) {
	source := `function sync(feed, api) { return "not an array of groups"; }`
	_, err := Run(context.Background(), source, Feed{}, testLimits())
	if err == nil {
		t.Fatal("Run() error = nil, want error (malformed return shape)")
	}
}

func TestRunEmptyCategoryOrService(t *testing.T) {
	source := `function sync(feed, api) { return [{category: "", service: "X", filters: []}]; }`
	_, err := Run(context.Background(), source, Feed{}, testLimits())
	if err == nil {
		t.Fatal("Run() error = nil, want error (empty category)")
	}
}

func TestRunInvalidEntryType(t *testing.T) {
	source := `function sync(feed, api) { return [{category: "C", service: "S", filters: [{type: "bogus", value: "x"}]}]; }`
	_, err := Run(context.Background(), source, Feed{}, testLimits())
	if err == nil {
		t.Fatal("Run() error = nil, want error (invalid entry type)")
	}
}

func TestRunEmptyEntryValue(t *testing.T) {
	source := `function sync(feed, api) { return [{category: "C", service: "S", filters: [{type: "domain", value: ""}]}]; }`
	_, err := Run(context.Background(), source, Feed{}, testLimits())
	if err == nil {
		t.Fatal("Run() error = nil, want error (empty entry value)")
	}
}

func TestRunMaxEntries(t *testing.T) {
	limits := testLimits()
	limits.MaxEntries = 3
	source := `
function sync(feed, api) {
    var filters = [];
    for (var i = 0; i < 10; i++) {
        filters.push({type: "domain", value: "host" + i + ".example.com"});
    }
    return [{category: "C", service: "S", filters: filters}];
}
`
	_, err := Run(context.Background(), source, Feed{}, limits)
	if err == nil {
		t.Fatal("Run() error = nil, want error (exceeds max entries)")
	}
}

func TestRunAPILogDoesNotError(t *testing.T) {
	source := `
function sync(feed, api) {
    api.log("hello from adapter");
    return [];
}
`
	_, err := Run(context.Background(), source, Feed{}, testLimits())
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
}
