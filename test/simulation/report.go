// Copyright (c) 2021 - 2026, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

//go:build simulation

package simulation

import (
	"encoding/json"
	"fmt"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"
)

// Outcome is one observable effect of a release: a push to a client, or a
// Discord notification about one.
type Outcome struct {
	Channel string `json:"channel"`
	Key     string `json:"key"`
}

// Step is one filter catching a release on its way through autobrr.
type Step struct {
	Filter   string `json:"filter"`
	Client   Client `json:"client"`
	Category string `json:"category,omitempty"`
	Rejected bool   `json:"rejected"`
}

// Expect walks the filters in priority order the way autobrr does: the first
// matching filter wins, unless its arr rejects the release, in which case the
// next matching filter gets a turn.
func (s *Scenario) Expect(rls *Release) []Step {
	filters := slices.Clone(s.Filters)
	slices.SortStableFunc(filters, func(a, b *FilterSpec) int { return int(b.Priority - a.Priority) })

	var steps []Step

	for _, f := range filters {
		if !slices.Contains(f.Indexers, rls.Indexer.Key) || !f.Matches(rls) {
			continue
		}

		rejected := f.Client.isArr() && rls.Rejected

		steps = append(steps, Step{Filter: f.Name, Client: f.Client, Category: f.Category, Rejected: rejected})

		if !rejected {
			break
		}
	}

	return steps
}

func (s *Scenario) filterByName(name string) *FilterSpec {
	for _, f := range s.Filters {
		if f.Name == name {
			return f
		}
	}

	return nil
}

func expectedOutcomes(s *Scenario, rls *Release) []Outcome {
	var out []Outcome

	for _, step := range s.Expect(rls) {
		action := s.filterByName(step.Filter).ActionName()

		switch step.Client {
		case ClientQbittorrent:
			out = append(out, Outcome{Channel: "qbittorrent", Key: qbitKey(rls.Name, step.Category)})
		default:
			out = append(out, Outcome{Channel: string(step.Client), Key: arrKey(rls.Name, !step.Rejected)})
		}

		event := "approved"
		if step.Rejected {
			event = "rejected"
		}

		out = append(out, Outcome{Channel: "discord", Key: discordKey(event, rls.Name, action)})
	}

	return out
}

func qbitKey(name, category string) string {
	return fmt.Sprintf("%s [%s]", name, category)
}

func arrKey(name string, approved bool) string {
	if approved {
		return name + " (approved)"
	}

	return name + " (rejected)"
}

func discordKey(event, name, action string) string {
	return fmt.Sprintf("%s: %s via %s", event, name, action)
}

// observed collects every outcome the mocks have seen, along with when.
func (s *Scenario) observed() ([]Outcome, map[string]time.Time) {
	var out []Outcome
	at := map[string]time.Time{}

	add := func(o Outcome, t time.Time, name string) {
		out = append(out, o)

		if prev, ok := at[name+"|"+o.Channel]; !ok || t.Before(prev) {
			at[name+"|"+o.Channel] = t
		}
	}

	for _, p := range s.Qbittorrent.Pushes() {
		key := qbitKey(p.Name, p.Category)
		if p.Error != "" {
			key = "error: " + p.Error
		}

		add(Outcome{Channel: "qbittorrent", Key: key}, p.At, p.Name)
	}

	for client, arr := range map[Client]*Arr{ClientSonarr: s.Sonarr, ClientRadarr: s.Radarr} {
		for _, p := range arr.Pushes() {
			key := arrKey(p.Name, p.Approved)
			if p.Error != "" {
				key = p.Name + " (error: " + p.Error + ")"
			}

			add(Outcome{Channel: string(client), Key: key}, p.At, p.Name)
		}
	}

	for _, msg := range s.Discord.Messages() {
		for _, embed := range msg.Message.Embeds {
			event := "unknown"
			if embed.Author != nil {
				event = strings.TrimPrefix(embed.Author.Name, "autobrr: release ")
			}

			var name, action string
			for _, field := range embed.Fields {
				switch field.Name {
				case "Release":
					name = field.Value
				case "Action":
					action = field.Value
				}
			}

			add(Outcome{Channel: "discord", Key: discordKey(event, name, action)}, msg.ReceivedAt, name)
		}
	}

	return out, at
}

// diff returns the outcomes only in want and only in got, as multisets.
func diff(want, got []Outcome) (missing, unexpected []Outcome) {
	counts := map[Outcome]int{}
	for _, o := range got {
		counts[o]++
	}

	for _, o := range want {
		if counts[o] > 0 {
			counts[o]--
			continue
		}

		missing = append(missing, o)
	}

	for _, o := range got {
		if counts[o] > 0 {
			counts[o]--
			unexpected = append(unexpected, o)
		}
	}

	sortOutcomes(missing)
	sortOutcomes(unexpected)

	return missing, unexpected
}

func sortOutcomes(o []Outcome) {
	slices.SortFunc(o, func(a, b Outcome) int {
		return strings.Compare(a.Channel+a.Key, b.Channel+b.Key)
	})
}

// Report is the result of one simulation run.
type Report struct {
	Seed       uint64        `json:"seed"`
	Releases   int           `json:"releases"`
	Duration   time.Duration `json:"duration_ns"`
	Injection  time.Duration `json:"injection_ns"`
	Settled    bool          `json:"settled"`
	AppExit    string        `json:"app_exit,omitempty"`
	Expected   int           `json:"expected_outcomes"`
	Observed   int           `json:"observed_outcomes"`
	Missing    []Outcome     `json:"missing"`
	Unexpected []Outcome     `json:"unexpected"`
	Warnings   []string      `json:"warnings"`

	ByKind     map[Kind]int              `json:"by_kind"`
	ByIndexer  map[string]int            `json:"by_indexer"`
	ByChannel  map[string][2]int         `json:"by_channel"`
	Latency    map[string]LatencySummary `json:"latency"`
	PerRelease []ReleaseResult           `json:"per_release"`
}

// LatencySummary is how long it took from delivering a release to its first push.
type LatencySummary struct {
	Count int           `json:"count"`
	P50   time.Duration `json:"p50_ns"`
	P95   time.Duration `json:"p95_ns"`
	Max   time.Duration `json:"max_ns"`
}

// ReleaseResult is the ground truth and expectation for a single release.
type ReleaseResult struct {
	ID       int    `json:"id"`
	Name     string `json:"name"`
	Kind     Kind   `json:"kind"`
	Indexer  string `json:"indexer"`
	Rejected bool   `json:"arr_rejects"`
	Steps    []Step `json:"steps"`
}

// Failed reports whether autobrr's behaviour deviated from the expectation.
func (r *Report) Failed() bool {
	return len(r.Missing) > 0 || len(r.Unexpected) > 0 || !r.Settled || r.AppExit != ""
}

// Verify waits until the mocks have seen at least the expected outcomes or the
// timeout passes, then waits settle more for stragglers such as duplicate
// pushes, and diffs the expectation against what was observed.
func (s *Scenario) Verify(releases []*Release, injected map[int]time.Time, timeout, settle time.Duration) *Report {
	var want []Outcome

	report := &Report{
		Releases:  len(releases),
		ByKind:    map[Kind]int{},
		ByIndexer: map[string]int{},
		ByChannel: map[string][2]int{},
		Latency:   map[string]LatencySummary{},
	}

	for _, rls := range releases {
		want = append(want, expectedOutcomes(s, rls)...)

		report.ByKind[rls.Kind]++
		report.ByIndexer[rls.Indexer.Name]++
		report.PerRelease = append(report.PerRelease, ReleaseResult{
			ID:       rls.ID,
			Name:     rls.Name,
			Kind:     rls.Kind,
			Indexer:  rls.Indexer.Name,
			Rejected: rls.Rejected,
			Steps:    s.Expect(rls),
		})
	}

	deadline := time.Now().Add(timeout)

	for {
		got, _ := s.observed()
		missing, _ := diff(want, got)

		if len(missing) == 0 {
			report.Settled = true
			break
		}

		if time.Now().After(deadline) {
			break
		}

		time.Sleep(250 * time.Millisecond)
	}

	time.Sleep(settle)

	got, firstSeen := s.observed()

	report.Expected = len(want)
	report.Observed = len(got)
	report.Missing, report.Unexpected = diff(want, got)

	for _, o := range want {
		c := report.ByChannel[o.Channel]
		c[0]++
		report.ByChannel[o.Channel] = c
	}

	for _, o := range got {
		c := report.ByChannel[o.Channel]
		c[1]++
		report.ByChannel[o.Channel] = c
	}

	latencies := map[string][]time.Duration{}

	for _, rls := range releases {
		start, ok := injected[rls.ID]
		if !ok {
			continue
		}

		// a release rejected by an arr falls through to another client, only its first push counts
		var first time.Time
		for _, channel := range []string{"qbittorrent", "sonarr", "radarr"} {
			if at, ok := firstSeen[rls.Name+"|"+channel]; ok && (first.IsZero() || at.Before(first)) {
				first = at
			}
		}

		if !first.IsZero() {
			source := string(rls.Indexer.Kind)
			latencies[source] = append(latencies[source], first.Sub(start))
		}
	}

	for source, l := range latencies {
		slices.Sort(l)

		report.Latency[source] = LatencySummary{
			Count: len(l),
			P50:   l[len(l)/2],
			P95:   l[min(len(l)-1, len(l)*95/100)],
			Max:   l[len(l)-1],
		}
	}

	for _, idx := range s.Indexers {
		for _, d := range idx.Tracker.Denied() {
			report.Warnings = append(report.Warnings, fmt.Sprintf("tracker %s denied %s", idx.Name, d))
		}

		for id, n := range idx.Tracker.Downloads() {
			if n > 2 {
				report.Warnings = append(report.Warnings, fmt.Sprintf("tracker %s served release %d %d times", idx.Name, id, n))
			}
		}
	}

	for name, unknown := range map[string][]string{"qbittorrent": s.Qbittorrent.Unknown(), "sonarr": s.Sonarr.Unknown(), "radarr": s.Radarr.Unknown()} {
		for _, u := range unknown {
			report.Warnings = append(report.Warnings, fmt.Sprintf("%s mock got unhandled request %s", name, u))
		}
	}

	slices.Sort(report.Warnings)
	report.Warnings = slices.Compact(report.Warnings)

	return report
}

// Markdown renders the report for a terminal or a GitHub job summary.
func (r *Report) Markdown() string {
	var b strings.Builder

	status := "PASS"
	if r.Failed() {
		status = "FAIL"
	}

	fmt.Fprintf(&b, "## autobrr simulation: %s\n\n", status)
	fmt.Fprintf(&b, "- seed: `%d` (rerun with `-seed=%d`)\n", r.Seed, r.Seed)
	fmt.Fprintf(&b, "- releases: %d, injected in %s, total %s\n", r.Releases, r.Injection.Round(time.Millisecond), r.Duration.Round(time.Millisecond))
	fmt.Fprintf(&b, "- outcomes: %d expected, %d observed, settled: %t\n", r.Expected, r.Observed, r.Settled)

	if r.AppExit != "" {
		fmt.Fprintf(&b, "- autobrr exited before the harness stopped it: `%s`\n", r.AppExit)
	}

	b.WriteString("\n")

	b.WriteString("| channel | expected | observed |\n|---|---:|---:|\n")
	for _, channel := range slices.Sorted(maps.Keys(r.ByChannel)) {
		c := r.ByChannel[channel]
		fmt.Fprintf(&b, "| %s | %d | %d |\n", channel, c[0], c[1])
	}

	b.WriteString("\n| release kind | count |\n|---|---:|\n")
	for _, kind := range slices.Sorted(maps.Keys(r.ByKind)) {
		fmt.Fprintf(&b, "| %s | %d |\n", kind, r.ByKind[kind])
	}

	b.WriteString("\n| indexer | releases |\n|---|---:|\n")
	for _, name := range slices.Sorted(maps.Keys(r.ByIndexer)) {
		fmt.Fprintf(&b, "| %s | %d |\n", name, r.ByIndexer[name])
	}

	b.WriteString("\n| source | pushes | p50 | p95 | max |\n|---|---:|---:|---:|---:|\n")
	for _, source := range slices.Sorted(maps.Keys(r.Latency)) {
		l := r.Latency[source]
		fmt.Fprintf(&b, "| %s | %d | %s | %s | %s |\n", source, l.Count, l.P50.Round(time.Millisecond), l.P95.Round(time.Millisecond), l.Max.Round(time.Millisecond))
	}

	writeOutcomes(&b, "Missing (expected, never observed)", r.Missing)
	writeOutcomes(&b, "Unexpected (observed, not expected)", r.Unexpected)

	if len(r.Warnings) > 0 {
		fmt.Fprintf(&b, "\n### Warnings (%d)\n\n", len(r.Warnings))
		for _, w := range r.Warnings[:min(len(r.Warnings), 50)] {
			fmt.Fprintf(&b, "- %s\n", w)
		}
	}

	return b.String()
}

func writeOutcomes(b *strings.Builder, title string, outcomes []Outcome) {
	if len(outcomes) == 0 {
		return
	}

	fmt.Fprintf(b, "\n### %s (%d)\n\n", title, len(outcomes))

	for _, o := range outcomes[:min(len(outcomes), 50)] {
		fmt.Fprintf(b, "- `%s` %s\n", o.Channel, o.Key)
	}

	if len(outcomes) > 50 {
		fmt.Fprintf(b, "- ... and %d more, see report.json\n", len(outcomes)-50)
	}
}

// Write saves report.md and report.json to dir, and appends the markdown to
// the GitHub job summary when running in Actions.
func (r *Report) Write(dir string) error {
	md := r.Markdown()

	if err := os.WriteFile(filepath.Join(dir, "report.md"), []byte(md), 0o644); err != nil {
		return err
	}

	data, err := json.MarshalIndent(r, "", "  ")
	if err != nil {
		return err
	}

	if err := os.WriteFile(filepath.Join(dir, "report.json"), data, 0o644); err != nil {
		return err
	}

	summary := os.Getenv("GITHUB_STEP_SUMMARY")
	if summary == "" {
		return nil
	}

	f, err := os.OpenFile(summary, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	defer f.Close()

	_, err = f.WriteString(md + "\n")

	return err
}
