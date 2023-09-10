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
	username   = "test"
	password   = "Test2023!"
	filterName = "Filter1"
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

	//time.Sleep(time.Millisecond * 2000)

	/* TODO Add indexer */

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
