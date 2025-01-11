// Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

//go:build e2e

package e2e_test

import (
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/playwright-community/playwright-go"
	"github.com/stretchr/testify/assert"
)

var (
	baseURL    = "http://127.0.0.1:7474"
	username   = "dev"
	password   = "pass"
	filterName = "Filter1"
	rssUrl     = "https://distrowatch.com/news/torrents.xml"
	rssKey     = "test-key-123"
	// Test data for notifications
	discordWebhook = "https://discord.com/api/webhooks/test"
	// Test data for IRC settings
	ircServer = "irc.libera.chat"
	ircNick   = "testuser"
	// Test data for API settings
	apiKey = "testapikey123"
	// Test data for application settings
	logLevel = "debug"
	// Test data for qBittorrent client
	qbitHost = "http://localhost:8080"
)

func TestEndToEnd(t *testing.T) {
	var (
		headless = true
	)

	if os.Getenv("HEADLESS") == "false" {
		headless = false
	}

	runOption := &playwright.RunOptions{
		Browsers: []string{"chromium"},
		//SkipInstallBrowsers: true,
	}

	if runtime.GOOS == "windows" {
		log.Println("Installing Playwright deps on Windows ..")

		if err := playwright.Install(runOption); err != nil {
			log.Fatalf("could not install Playwright: %v", err)
		}
	}

	t.Run("Health check", func(t *testing.T) {
		healthResp, err := http.Get(baseUrl("/api/healthz/liveness"))
		defer healthResp.Body.Close()

		assert.NoError(t, err, "could not get health check")
		assert.Equal(t, http.StatusOK, healthResp.StatusCode)
	})

	pw, err := playwright.Run(runOption)
	assert.NoError(t, err, "could not launch playwright")

	browser, err := pw.Chromium.Launch(playwright.BrowserTypeLaunchOptions{
		Headless: &headless,
	})
	assert.NoError(t, err, "could not launch Chromium")

	context, err := browser.NewContext()
	assert.NoError(t, err, "could not create context")

	page, err := context.NewPage()
	assert.NoError(t, err, "could not create page")

	_, err = page.Goto(baseUrl("/"))
	assert.NoError(t, err, "could not goto base url")

	// Run tests
	tests := []struct {
		name string
		fn   func(*testing.T, playwright.Page) error
	}{
		{"Register", testRegister},
		{"Login", testLogin},
		{"Add Indexer", testAddIndexer},
		{"Add MockIndexer", testAddMockIndexer},
		{"Add Notification", testNotifications},
		//{"Configure IRC", testIRCSettings},
		{"Enable Mock IRC", testIRCEnableMockIndexer},
		{"Configure API", testAPISettings},
		{"Configure Application", testApplicationSettings},
		{"Configure Download Clients", testDownloadClients},
		{"Add Filter", testAddFilter},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.fn(t, page)
			assert.NoError(t, err)
			//time.Sleep(time.Millisecond * 1000) // Wait between tests
		})
	}

	assert.NoErrorf(t, browser.Close(), "could not close browser")
	assert.NoErrorf(t, pw.Stop(), "could not stop Playwright")
}

func baseUrl(endpoint string) string {
	b, _ := url.JoinPath(baseURL, endpoint)
	return b
}

// Helper function to fill input field
func fillInput(page playwright.Page, selector string, value string) error {
	if err := page.Locator(selector).Fill(value); err != nil {
		return fmt.Errorf("could not fill input %s: %v", selector, err)
	}
	return nil
}

// Helper function to click button with text
func clickButton(page playwright.Page, text string) error {
	if err := page.Locator("button", playwright.PageLocatorOptions{
		HasText: text,
	}).Click(); err != nil {
		return fmt.Errorf("could not click button with text '%s': %v", text, err)
	}
	return nil
}

// Helper function to select from dropdown
func selectFromDropdown(page playwright.Page, inputSelector string, value string) error {
	// Click to open the dropdown
	if err := page.Locator(inputSelector).Click(); err != nil {
		return fmt.Errorf("could not click dropdown %s: %v", inputSelector, err)
	}

	// Wait a bit for the dropdown to open
	time.Sleep(time.Millisecond * 500)

	// Click the option with the specified text
	optionSelector := fmt.Sprintf("div[role='option']:has-text('%s')", value)
	if err := page.Locator(optionSelector).Click(); err != nil {
		return fmt.Errorf("could not select option %s: %v", value, err)
	}

	return nil
}

// Helper function to wait for navigation and check if element exists
func waitForElement(page playwright.Page, selector string, timeout float64) error {
	return page.Locator(selector).WaitFor(playwright.LocatorWaitForOptions{
		State:   playwright.WaitForSelectorStateVisible,
		Timeout: playwright.Float(timeout),
	})
}

// Helper function to check if element contains text
func elementContainsText(page playwright.Page, selector string, text string) bool {
	element, err := page.QuerySelector(selector)
	if err != nil {
		return false
	}
	content, err := element.TextContent()
	if err != nil {
		return false
	}
	return strings.Contains(content, text)
}

func testRegister(t *testing.T, page playwright.Page) error {
	// Test register
	assert.NoErrorf(t, page.Locator("input#username").Fill(username), "could not fill input #username")
	assert.NoErrorf(t, page.Locator("input#password1").Fill(password), "could not fill input #password1")
	assert.NoErrorf(t, page.Locator("input#password2").Fill(password), "could not fill input #password2")
	assert.NoErrorf(t, page.Locator("text=Create account").Click(), "could not click button with text 'Create account'")

	// Verify successful onboarding by checking for Login form
	assert.NoErrorf(t, waitForElement(page, "text=Sign in", 5000), "register failed: could not find login form")

	return nil
}

func testLogin(t *testing.T, page playwright.Page) error {
	// Test login
	assert.NoError(t, page.Locator("input#username").Fill(username), "could not fill input #username")
	assert.NoError(t, page.Locator("input#password").Fill(password), "could not fill input #password")
	assert.NoError(t, page.Locator("text=Sign in").Click(), "could not click button with text 'Sign in'")

	// Verify successful login by checking for Stats text
	err := waitForElement(page, "text=Stats", 5000)
	assert.NoError(t, err, "login failed: could not find stats text")

	return nil
}

func testAddIndexer(t *testing.T, page playwright.Page) error {
	// Navigate to indexers page
	_, err := page.Goto(baseUrl("/settings/indexers"))
	assert.NoError(t, err, "could not navigate to settings/indexers")

	// Add new indexer
	assert.NoError(t, page.Locator("button", playwright.PageLocatorOptions{HasText: "Add new indexer"}).Click(), "could not find and click 'Add new indexer' button")

	// Wait for and select indexer type
	assert.NoError(t, page.Locator("[id^='react-select'][id$='-input']").WaitFor(), "could not wait for indexer type dropdown")

	err = selectFromDropdown(page, "[id^='react-select'][id$='-input']", "Generic RSS")
	assert.NoError(t, err, "could not select indexer type")

	// Wait for and fill RSS URL
	assert.NoError(t, page.Locator("input#feed\\.url").WaitFor(), "could not find RSS URL input field")
	assert.NoError(t, page.Locator("input#feed\\.url").Fill(rssUrl), "could not type in RSS URL")

	//time.Sleep(time.Millisecond * 1000)

	// Save indexer
	assert.NoError(t, page.Locator("button[type='submit']").Click(), "could not click Save button")

	err = page.Locator("ul > li").GetByText("Generic RSS").WaitFor()
	assert.NoError(t, err, "could not find indexer in list")

	return nil
}

func testAddMockIndexer(t *testing.T, page playwright.Page) error {
	// Navigate to indexers page
	_, err := page.Goto(baseUrl("/settings/indexers"))
	assert.NoError(t, err, "could not navigate to settings/indexers")

	// Add new indexer
	assert.NoError(t, page.Locator("button", playwright.PageLocatorOptions{HasText: "Add new"}).Click(), "could not find and click 'Add new' button")

	// Wait for and select indexer type
	assert.NoError(t, page.Locator("[id^='react-select'][id$='-input']").WaitFor(), "could not wait for indexer type dropdown")

	//page.GetByRole()

	err = page.Locator("[id^='react-select'][id$='-input']").Click()
	assert.NoError(t, err, "could not click dropdown")

	// Wait a bit for the dropdown to open
	time.Sleep(time.Millisecond * 500)

	err = selectFromDropdown(page, "[id^='react-select'][id$='-input']", "MockIndexer")
	assert.NoError(t, err, "could not select indexer type")

	// Wait for and fill RSS key
	assert.NoError(t, page.Locator("input#settings\\.rsskey").WaitFor(), "could not find RSS key input field")
	assert.NoError(t, page.Locator("input#settings\\.rsskey").Fill(rssKey), "could not type in RSS key")

	assert.NoError(t, page.Locator("input#irc\\.nick").Fill(ircNick), "could not type in IRC nick")

	//time.Sleep(time.Millisecond * 1000)

	// Save indexer
	assert.NoError(t, page.Locator("button[type='submit']").Click(), "could not click Save button")

	err = page.Locator("ul > li").GetByText("MockIndexer").WaitFor()
	assert.NoError(t, err, "could not find indexer in list")

	return nil
}

func testAddFilter(t *testing.T, page playwright.Page) error {
	var err error

	// Navigate to filters page
	_, err = page.Goto(baseUrl("/filters"))
	assert.NoError(t, err, "could not navigate to filters page")

	// Add new filter
	err = page.Locator("text=Add new").Click()
	assert.NoError(t, err, "could not find and click 'Add new' button")

	// Fill filter name
	err = page.Locator("input#name").Fill(filterName)
	assert.NoError(t, err, "could not fill filter name")

	// Wait for form to be ready
	//time.Sleep(time.Millisecond * 2000)

	// Create filter
	err = page.Locator("button[type='submit']").Click()
	assert.NoError(t, err, "could not click Create button")

	_, err = page.Goto(baseUrl("/filters"))
	assert.NoError(t, err, "could not navigate to filters page")

	// Verify filter was created
	err = page.Locator("ul > li").GetByText(filterName).WaitFor()
	assert.NoError(t, err, "could not find filter in list")

	return nil
}

func testNotifications(t *testing.T, page playwright.Page) error {
	// Navigate to notifications settings
	_, err := page.Goto(baseUrl("/settings/notifications"))
	assert.NoError(t, err, "could not navigate to notifications settings")

	// Add new notification and wait for slide-over transition
	assert.NoError(t, page.Locator("button", playwright.PageLocatorOptions{HasText: "Add new"}).Click(), "could not find and click 'Add new' button")

	// Wait for slide-over transition to complete
	time.Sleep(time.Millisecond * 700)

	// Wait for and select Discord from dropdown
	assert.NoError(t, page.Locator("[id^='react-select'][id$='-input']").WaitFor(), "could not find notification type dropdown")

	err = selectFromDropdown(page, "[id^='react-select'][id$='-input']", "Discord")
	assert.NoError(t, err, "could not select notification type")

	// Wait for Discord form fields to appear
	time.Sleep(time.Millisecond * 500)

	// Fill notification name
	assert.NoError(t, page.Locator("input#name").Fill("Discord Test"), "could not fill name")

	// Fill webhook URL
	assert.NoError(t, page.Locator("input#webhook").Fill(discordWebhook), "could not fill webhook URL")

	// Select an event (using the first event checkbox)
	assert.NoError(t, page.Locator("input#events-PUSH_APPROVED").Click(), "could not select event")

	// Wait for form to be ready
	time.Sleep(time.Millisecond * 500)

	// Save notification
	assert.NoError(t, page.Locator("button[type='submit']").Click(), "could not click Save button")

	// Verify notification was added
	err = page.Locator("ul > li").GetByText("Discord Test").WaitFor()
	assert.NoError(t, err, "could not find notification in list")

	return nil
}

func testIRCSettings(t *testing.T, page playwright.Page) error {
	// Navigate to IRC settings
	_, err := page.Goto(baseUrl("/settings/irc"))
	assert.NoErrorf(t, err, "could not navigate to IRC settings")

	// Add new IRC network
	assert.NoErrorf(t, page.Locator("button", playwright.PageLocatorOptions{HasText: "Add new"}).Click(), "could not find and click 'Add new' button")

	// Wait for form to be ready
	time.Sleep(time.Millisecond * 1000)

	// Fill IRC settings
	assert.NoError(t, page.Locator("input#name").Fill("Test Network"), "could not fill name")
	assert.NoError(t, page.Locator("input#server").Fill(ircServer), "could not fill server")
	assert.NoError(t, page.Locator("input#nick").Fill(ircNick), "could not fill nick")
	assert.NoError(t, page.Locator("input#auth\\.account").Fill(ircNick), "could not fill auth account")
	assert.NoError(t, page.Locator("input#auth\\.password").Fill("testpass"), "could not fill auth password")

	// Uncheck enabled switch (it's true by default)
	assert.NoError(t, page.Locator("form > button#enabled").Click(), "could not click disabled switch")

	// Wait for switch state to be updated
	time.Sleep(time.Millisecond * 500)

	// Create IRC network
	assert.NoError(t, page.Locator("button", playwright.PageLocatorOptions{HasText: "Create"}).Click(), "could not click Create button")

	// Verify IRC network was added
	err = page.Locator("ul > li").GetByText(ircServer).WaitFor()
	assert.NoError(t, err, "could not find IRC network in list")

	return nil
}

func testIRCEnableMockIndexer(t *testing.T, page playwright.Page) error {
	// Navigate to IRC settings
	_, err := page.Goto(baseUrl("/settings/irc"))
	assert.NoErrorf(t, err, "could not navigate to IRC settings")

	// Verify IRC network was added
	err = page.Locator("ul > li").GetByText("Mock").WaitFor()
	assert.NoError(t, err, "could not find IRC network in list")

	err = page.Locator("ul > li").Locator("button#enabled").Click()
	assert.NoError(t, err, "could not click enabled switch")

	return nil
}

func testAPISettings(t *testing.T, page playwright.Page) error {
	// Navigate to API settings
	_, err := page.Goto(baseUrl("/settings/api"))
	assert.NoError(t, err, "could not navigate to API settings")

	// Add new API key and wait for slide-over transition
	assert.NoError(t, page.Locator("button", playwright.PageLocatorOptions{HasText: "Add new"}).Click(), "could not find and click 'Add new' button")

	// Wait for slide-over transition to complete
	time.Sleep(time.Millisecond * 700)

	// Fill API key name
	assert.NoError(t, page.Locator("input#name").Fill(apiKey), "could not fill API key name")

	// Create API key (using specific class to target the correct button)
	assert.NoError(t, page.Locator("button.bg-blue-600[type='submit']").Click(), "could not click Create button")

	// Verify API key was added
	err = page.Locator("ul > li").GetByText(apiKey).WaitFor()
	assert.NoError(t, err, "could not find API key in list")

	return nil
}

func testApplicationSettings(t *testing.T, page playwright.Page) error {
	// Navigate to logs settings
	_, err := page.Goto(baseUrl("/settings/logs"))
	assert.NoError(t, err, "could not navigate to logs settings")

	// Wait for and change log level
	assert.NoError(t, page.Locator("[id^='react-select'][id$='-input']").WaitFor(), "could not find log level dropdown")

	assert.NoError(t, selectFromDropdown(page, "[id^='react-select'][id$='-input']", "Debug"), "could not select log level")

	// Wait for settings to be saved automatically
	err = waitForElement(page, "text=Config successfully updated!", 5000)
	assert.NoError(t, err, "log settings were not saved successfully")

	return nil
}

// Helper function to add a download client and return any error
func addDownloadClient(t *testing.T, page playwright.Page, name, host string) error {
	// Add new client and wait for slide-over animation
	err := clickButton(page, "Add new client")
	assert.NoError(t, err, "could not find and click 'Add new client' button")

	// Wait for slide-over animation to complete
	time.Sleep(time.Millisecond * 700)

	// Wait for form to be ready and fill name
	assert.NoError(t, page.Locator("input#name").WaitFor(), "could not find input name")

	err = fillInput(page, "input#name", name)
	assert.NoError(t, err, "could not fill input name")

	// Wait for and fill host field
	assert.NoError(t, page.Locator("input#host").WaitFor(), "could not find input host")

	err = fillInput(page, "input#host", host)
	assert.NoError(t, err, "could not fill input host")

	// Wait for form validation
	time.Sleep(time.Millisecond * 500)

	// Create client
	err = page.Locator("button[type='submit']").Click()
	assert.NoError(t, err, "could not click Create button")

	// Verify client was added
	err = page.Locator("ul > li").GetByText(name).WaitFor()
	assert.NoError(t, err, "could not find Download Client in list")

	return nil
}

func testDownloadClients(t *testing.T, page playwright.Page) error {
	// Navigate to download clients settings
	_, err := page.Goto(baseUrl("/settings/clients"))
	assert.NoError(t, err, "could not navigate to download clients settings")

	// Add qBittorrent client
	err = addDownloadClient(t, page, "qbit-test", qbitHost)
	assert.NoError(t, err, "could not add qBittorrent client")

	return nil
}
