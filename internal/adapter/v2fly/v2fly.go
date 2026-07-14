// Package v2fly parses the per-service domain list format used by
// v2fly/domain-list-community (github.com/v2fly/domain-list-community),
// the canonical upstream source for per-service domain data (see project
// memory: researched 2026-07-14).
//
// Each line is one of:
//
//	full:example.com     exact match
//	domain:example.com   suffix match (example.com and *.example.com)
//	keyword:example      substring match
//	example.com          bare domain, same as domain:example.com
//	regexp:^example\.    regex match — not supported, see below
//	include:other-file   pulls in another file — not supported, see below
//
// A trailing "@attribute" (e.g. "@cn", "@ads") tags a line within its file
// and is stripped along with anything after it; "#" starts a comment,
// whether the whole line or trailing after content.
//
// regexp: and include: lines are skipped rather than erroring:
// regexp doesn't map to any of our EntryTypes (MikroTik's DNS static/
// address-list mechanism has no regex matching to target it at), and
// include pulls in a whole separate file, which would turn "fetch one
// URL" into "fetch a dependency graph of files" — deferred until a real
// feed actually needs it.
package v2fly

import (
	"bufio"
	"fmt"
	"io"
	"strings"

	"github.com/andrey-vk/wdsplit/internal/adapter"
	"github.com/andrey-vk/wdsplit/internal/model"
)

// Parser parses the v2fly domain-list-community line format.
type Parser struct{}

func (Parser) Parse(r io.Reader) ([]adapter.Entry, error) {
	var entries []adapter.Entry

	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		line := scanner.Text()

		if idx := strings.IndexByte(line, '#'); idx >= 0 {
			line = line[:idx]
		}
		if idx := strings.IndexByte(line, '@'); idx >= 0 {
			line = line[:idx]
		}
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		switch {
		case strings.HasPrefix(line, "include:"):
			continue
		case strings.HasPrefix(line, "regexp:"):
			continue
		case strings.HasPrefix(line, "full:"):
			entries = append(entries, adapter.Entry{
				Type:  model.EntryDomain,
				Value: strings.TrimPrefix(line, "full:"),
			})
		case strings.HasPrefix(line, "domain:"):
			entries = append(entries, adapter.Entry{
				Type:  model.EntryDomainSuffix,
				Value: strings.TrimPrefix(line, "domain:"),
			})
		case strings.HasPrefix(line, "keyword:"):
			entries = append(entries, adapter.Entry{
				Type:  model.EntryDomainKeyword,
				Value: strings.TrimPrefix(line, "keyword:"),
			})
		default:
			entries = append(entries, adapter.Entry{
				Type:  model.EntryDomainSuffix,
				Value: line,
			})
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("scan v2fly domain list: %w", err)
	}

	return entries, nil
}
