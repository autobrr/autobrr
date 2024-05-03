package main

import (
	"fmt"
	"log"
	"reflect"
	"time"

	"github.com/playwright-community/playwright-go"
)

func assertErrorToNilf(message string, err error) {
	if err != nil {
		log.Fatalf(message, err)
	}
}

func assertBool(message string, actual, expect bool) {
	if actual != expect {
		log.Fatalf(message, actual)
	}
}

func assertEqual(expected, actual interface{}) {
	if !reflect.DeepEqual(expected, actual) {
		panic(fmt.Sprintf("%v does not equal %v", actual, expected))
	}
}

var (
	username   = "dev"
	password   = "pass"
	filterName = "Filter1"
	rssUrl     = "https://distrowatch.com/news/torrents.xml"
)

func main() {
	pw, err := playwright.Run()
	assertErrorToNilf("could not launch playwright: %w", err)

	browser, err := pw.Chromium.Launch(playwright.BrowserTypeLaunchOptions{
		Headless: playwright.Bool(false),
	})
	assertErrorToNilf("could not launch Chromium: %w", err)

	context, err := browser.NewContext()
	assertErrorToNilf("could not create context: %w", err)

	page, err := context.NewPage()
	assertErrorToNilf("could not create page: %w", err)

	_, err = page.Goto("http://localhost:3000")
	assertErrorToNilf("could not goto: %w", err)

	page.Locator("h1").IsVisible()

	assertErrorToNilf("could not type: %v", page.Locator("input#username").Fill(username))
	assertErrorToNilf("could not type: %v", page.Locator("input#password").Fill(password))

	assertErrorToNilf("could not press: %v", page.Locator("text=Sign in").Click())

	page.Locator("text=Stats").IsVisible()

	time.Sleep(time.Millisecond * 1000)

	/* Add indexer */
	_, err = page.Goto("http://localhost:3000/settings/indexers")
	assertErrorToNilf("could not navigate to settings/indexers: %w", err)

	// Wait for the button to be visible and then click it to add a new indexer
	assertErrorToNilf("could not find and click 'Add new indexer' button: %v", page.Locator("button", playwright.PageLocatorOptions{
		HasText: "Add new indexer",
	}).Click())

	assertErrorToNilf("could not fill 'Generic RSS' in the dropdown: %v", page.Locator("input#react-select-3-input").Fill("Generic RSS"))

	assertErrorToNilf("could not select 'Generic RSS' by pressing enter: %v", page.Locator("input#react-select-3-input").Press("Enter"))

	// Wait for the RSS URL input field to be visible
	assertErrorToNilf("could not find RSS URL input field: %v", page.Locator("input#feed\\.url").WaitFor())

	assertErrorToNilf("could not type in RSS URL: %v", page.Locator("input#feed\\.url").Fill(rssUrl))

	time.Sleep(time.Millisecond * 1000)

	assertErrorToNilf("could not click the 'Save' button: %v", page.Locator("button", playwright.PageLocatorOptions{
		HasText: "Save",
	}).Click())

	time.Sleep(time.Millisecond * 1000)

	/* Enable Feed */
	_, err = page.Goto("http://localhost:3000/settings/feeds")
	assertErrorToNilf("could not navigate to feeds settings: %w", err)

	switchButtonLocator := page.Locator("button[role='switch']")

	// Wait for the switch button to be visible
	assertErrorToNilf("could not find the switch button: %v", switchButtonLocator.WaitFor())

	assertErrorToNilf("could not click the switch button: %v", switchButtonLocator.Click())

	time.Sleep(time.Millisecond * 1000)

	/* Add Download Client */

	/* Add filter */

	assertErrorToNilf("could not press: %v", page.Locator("text=Filters").Click())

	//filterPageVisible, err := page.Locator("h1").IsVisible()
	//assertErrorToNilf("could not goto: %w", err)
	//
	//assertBool("", filterPageVisible, true)

	assertErrorToNilf("could not press: %v", page.Locator("text=Add Filter").Click())
	assertErrorToNilf("could not type: %v", page.Locator("input#name").Fill(filterName))
	assertErrorToNilf("could not press: %v", page.Locator("button", playwright.PageLocatorOptions{HasText: "Create"}).Click())

	time.Sleep(time.Millisecond * 2000)

	assertErrorToNilf("could not close browser: %w", browser.Close())
	assertErrorToNilf("could not stop Playwright: %w", pw.Stop())
}
