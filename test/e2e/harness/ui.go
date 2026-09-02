// Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

//go:build e2e

package harness

import (
	"fmt"
	"testing"

	"github.com/mxschmitt/playwright-go"
)

// UI wraps a Playwright page with the interactions the autobrr web UI needs.
//
// Every method fails the test on error rather than returning one. An e2e step
// has no meaningful recovery: if a form did not open there is nothing sensible
// to assert next, and continuing only turns one real failure into a page of
// noise. The embedded Page is still there for anything ad hoc.
//
// None of these methods sleep. Playwright locators already wait for an element
// to be attached, visible, stable and able to receive events, so a fixed sleep
// only makes a passing test slower and a failing one flakier.
type UI struct {
	playwright.Page

	t testing.TB
}

// Open navigates to a path relative to the instance's base URL.
func (u *UI) Open(path string) {
	u.t.Helper()

	if _, err := u.Goto(path); err != nil {
		u.t.Fatalf("could not open %s: %v", path, err)
	}
}

// AddNew clicks the "Add new" button in a settings section header. Exact
// matching keeps it off the empty-state button, which is captioned with the
// item type ("Add new indexer") and opens the same form.
func (u *UI) AddNew() {
	u.t.Helper()

	button := u.GetByRole("button", playwright.PageGetByRoleOptions{
		Name:  "Add new",
		Exact: new(true),
	})

	if err := button.First().Click(); err != nil {
		u.t.Fatalf("could not click 'Add new': %v", err)
	}
}

// Submit submits the form that is currently open.
func (u *UI) Submit() {
	u.t.Helper()

	if err := u.Locator("button[type='submit']").First().Click(); err != nil {
		u.t.Fatalf("could not submit form: %v", err)
	}
}

// ClickButton clicks a button by its accessible name.
func (u *UI) ClickButton(name string) {
	u.t.Helper()

	button := u.GetByRole("button", playwright.PageGetByRoleOptions{Name: name})

	if err := button.First().Click(); err != nil {
		u.t.Fatalf("could not click button %q: %v", name, err)
	}
}

// Fill types value into the field with the given id. Ids come from the shared
// input components, which set id and name from the Formik field name, so this
// is stable in a way that label text is not. It matches on the id alone rather
// than input#id because several fields (a filter's match rules, a watch folder
// path) are auto-resizing textareas.
func (u *UI) Fill(id, value string) {
	u.t.Helper()

	if err := u.Locator("#" + escapeID(id)).Fill(value); err != nil {
		u.t.Fatalf("could not fill %q: %v", id, err)
	}
}

// Toggle flips a Checkbox by its name. The component renders a headlessui
// Switch, so the element is a button rather than a checkbox input.
func (u *UI) Toggle(name string) {
	u.t.Helper()

	if err := u.Locator("button#" + escapeID(name)).First().Click(); err != nil {
		u.t.Fatalf("could not toggle %q: %v", name, err)
	}
}

// Select picks an option from the react-select dropdown in the open form. The
// forms only ever have one, so it does not need identifying further.
func (u *UI) Select(option string) {
	u.t.Helper()

	input := u.Locator("[id^='react-select'][id$='-input']").First()

	if err := input.Click(); err != nil {
		u.t.Fatalf("could not open dropdown to select %q: %v", option, err)
	}

	// Waiting on the option itself is what replaces the old fixed sleep: the
	// locator resolves once the menu has rendered it.
	item := u.Locator(fmt.Sprintf("div[role='option']:has-text(%q)", option))

	if err := item.First().Click(); err != nil {
		u.t.Fatalf("could not select %q: %v", option, err)
	}
}

// SelectListbox picks an option from a headlessui Listbox. That is a third
// dropdown implementation, distinct from the react-select one Select drives and
// the react-multi-select one SelectMulti drives, and it is what the filter
// action forms use.
func (u *UI) SelectListbox(option string) {
	u.t.Helper()

	if err := u.Locator("button[aria-haspopup='listbox']").First().Click(); err != nil {
		u.t.Fatalf("could not open listbox to select %q: %v", option, err)
	}

	item := u.GetByRole("option", playwright.PageGetByRoleOptions{Name: option, Exact: new(true)})

	if err := item.First().Click(); err != nil {
		u.t.Fatalf("could not select %q from the listbox: %v", option, err)
	}
}

// SelectMulti ticks an option in a react-multi-select-component dropdown, which
// is what the filter form uses to pick indexers. It is a different widget from
// the react-select one Select drives, with its own markup.
//
// field names which dropdown to use. The filter's General tab has two of them
// side by side, so an unscoped lookup would open whichever came first.
func (u *UI) SelectMulti(field, option string) {
	u.t.Helper()

	dropdown := u.Locator(fmt.Sprintf("[aria-labelledby=%q]", field))

	if err := dropdown.Locator(".dropdown-heading").Click(); err != nil {
		u.t.Fatalf("could not open the %q multi-select: %v", field, err)
	}

	item := dropdown.Locator(fmt.Sprintf(".select-item:has-text(%q)", option))

	if err := item.First().Click(); err != nil {
		u.t.Fatalf("could not pick %q from the %q multi-select: %v", option, field, err)
	}

	// Close the dropdown again so its panel stops covering the fields below.
	if err := u.Keyboard().Press("Escape"); err != nil {
		u.t.Fatalf("could not close the %q multi-select: %v", field, err)
	}
}

// Expand opens a collapsible filter section so the field inside it becomes
// reachable. The filter tabs render their sections collapsed and drop the
// children entirely while closed, so field is also how this tells an
// already-open section from a closed one: clicking unconditionally would
// collapse a section that had opened itself.
func (u *UI) Expand(title, field string) {
	u.t.Helper()

	count, err := u.Locator("#" + escapeID(field)).Count()
	if err != nil {
		u.t.Fatalf("could not look for %q: %v", field, err)
	}

	if count > 0 {
		return
	}

	heading := u.GetByRole("heading", playwright.PageGetByRoleOptions{Name: title, Exact: new(true)})

	if err := heading.First().Click(); err != nil {
		u.t.Fatalf("could not expand section %q: %v", title, err)
	}
}

// ClickLink follows an in-page link by its accessible name. Filter tabs have to
// be reached this way rather than by URL: the tabs share one Formik form, and a
// fresh page load would discard everything typed so far.
func (u *UI) ClickLink(name string) {
	u.t.Helper()

	link := u.GetByRole("link", playwright.PageGetByRoleOptions{Name: name, Exact: new(true)})

	if err := link.First().Click(); err != nil {
		u.t.Fatalf("could not click link %q: %v", name, err)
	}
}

// ExpectRow waits for a list row containing text. Every settings screen and the
// filter list render their items as `ul > li`.
func (u *UI) ExpectRow(text string) {
	u.t.Helper()

	row := u.Locator("ul > li").GetByText(text).First()

	if err := row.WaitFor(); err != nil {
		u.t.Fatalf("could not find %q in the list: %v", text, err)
	}
}

// ExpectText waits for text to appear anywhere on the page.
func (u *UI) ExpectText(text string) {
	u.t.Helper()

	if err := u.GetByText(text).First().WaitFor(); err != nil {
		u.t.Fatalf("could not find text %q on the page: %v", text, err)
	}
}

// escapeID escapes the characters TanStack Form puts in nested field names
// ("feed.url", "actions[0].watch_folder"), which CSS would otherwise read as
// class or attribute selectors.
func escapeID(id string) string {
	escaped := make([]rune, 0, len(id)+4)

	for _, r := range id {
		switch r {
		case '.', ':', '[', ']':
			escaped = append(escaped, '\\')
		}

		escaped = append(escaped, r)
	}

	return string(escaped)
}
