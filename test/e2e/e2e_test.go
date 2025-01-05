//go:build e2e
// +build e2e

package e2e_test

import (
	"fmt"
	"log"
	"net/url"
	"os"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/playwright-community/playwright-go"
)

var (
	baseURL    = "http://127.0.0.1:7474"
	username   = "dev"
	password   = "pass"
	filterName = "Filter1"
	rssUrl     = "https://distrowatch.com/news/torrents.xml"
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

	pw, err := playwright.Run()
	assertErrorToNilf("could not launch playwright: %w", err)

	browser, err := pw.Chromium.Launch(playwright.BrowserTypeLaunchOptions{
		Headless: &headless,
	})
	assertErrorToNilf("could not launch Chromium: %w", err)

	context, err := browser.NewContext()
	assertErrorToNilf("could not create context: %w", err)

	page, err := context.NewPage()
	assertErrorToNilf("could not create page: %w", err)

	res, err := page.Goto(baseUrl("/"))
	assertErrorToNilf("could not goto: %w", err)
	log.Printf("status code: %d status: %s\n", res.Status(), res.StatusText())
	log.Println(res.Text())

	// Run tests
	tests := []struct {
		name string
		fn   func(playwright.Page) error
	}{
		{"Register", testRegister},
		{"Login", testLogin},
		{"Add Indexer", testAddIndexer},
		{"Add Filter", testAddFilter},
		{"Add Notification", testNotifications},
		{"Configure IRC", testIRCSettings},
		{"Configure API", testAPISettings},
		{"Configure Application", testApplicationSettings},
		{"Configure Download Clients", testDownloadClients},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Logf("Running test: %s\n", tt.name)
			if err := tt.fn(page); err != nil {
				t.Errorf("Test %s failed: %v\n", tt.name, err)
			} else {
				t.Logf("Test %s passed\n", tt.name)
			}
			time.Sleep(time.Millisecond * 1000) // Wait between tests
		})
	}

	assertErrorToNilf("could not close browser: %w", browser.Close())
	assertErrorToNilf("could not stop Playwright: %w", pw.Stop())
}

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
	_, err := page.WaitForSelector(selector, playwright.PageWaitForSelectorOptions{
		State:   playwright.WaitForSelectorStateVisible,
		Timeout: playwright.Float(timeout),
	})
	return err
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

func testRegister(page playwright.Page) error {
	// Test register
	assertErrorToNilf("could not type: %v", page.Locator("input#username").Fill(username))
	assertErrorToNilf("could not type: %v", page.Locator("input#password1").Fill(password))
	assertErrorToNilf("could not type: %v", page.Locator("input#password2").Fill(password))
	assertErrorToNilf("could not press: %v", page.Locator("text=Create account").Click())

	// Verify successful onboarding by checking for Login form
	err := waitForElement(page, "text=Sign in", 5000)
	if err != nil {
		return fmt.Errorf("register failed: %v", err)
	}
	return nil
}

func testLogin(page playwright.Page) error {
	// Test login
	assertErrorToNilf("could not type: %v", page.Locator("input#username").Fill(username))
	assertErrorToNilf("could not type: %v", page.Locator("input#password").Fill(password))
	assertErrorToNilf("could not press: %v", page.Locator("text=Sign in").Click())

	// Verify successful login by checking for Stats text
	err := waitForElement(page, "text=Stats", 5000)
	if err != nil {
		return fmt.Errorf("login failed: %v", err)
	}
	return nil
}

func testAddIndexer(page playwright.Page) error {
	// Navigate to indexers page
	_, err := page.Goto(baseUrl("/settings/indexers"))
	if err != nil {
		return fmt.Errorf("could not navigate to settings/indexers: %v", err)
	}

	// Add new indexer
	assertErrorToNilf("could not find and click 'Add new indexer' button: %v", page.Locator("button", playwright.PageLocatorOptions{
		HasText: "Add new indexer",
	}).Click())

	// Wait for and select indexer type
	assertErrorToNilf("could not wait for indexer type dropdown: %v", page.Locator("[id^='react-select'][id$='-input']").WaitFor())
	if err := selectFromDropdown(page, "[id^='react-select'][id$='-input']", "Generic RSS"); err != nil {
		return fmt.Errorf("could not select indexer type: %v", err)
	}

	// Wait for and fill RSS URL
	assertErrorToNilf("could not find RSS URL input field: %v", page.Locator("input#feed\\.url").WaitFor())
	assertErrorToNilf("could not type in RSS URL: %v", page.Locator("input#feed\\.url").Fill(rssUrl))

	time.Sleep(time.Millisecond * 1000)

	// Save indexer
	assertErrorToNilf("could not click Save button: %v", page.Locator("button[type='submit']").Click())

	// Verify indexer was added
	err = waitForElement(page, "text=Generic RSS", 5000)
	if err != nil {
		return fmt.Errorf("indexer was not added successfully: %v", err)
	}
	return nil
}

func testAddFilter(page playwright.Page) error {
	var err error

	// Navigate to filters page
	if _, err = page.Goto(baseUrl("/filters")); err != nil {
		return fmt.Errorf("could not navigate to filters: %v", err)
	}

	// Add new filter
	if err = page.Locator("text=Add new").Click(); err != nil {
		return fmt.Errorf("could not click Add new button: %v", err)
	}

	// Fill filter name
	if err = page.Locator("input#name").Fill(filterName); err != nil {
		return fmt.Errorf("could not fill filter name: %v", err)
	}

	// Wait for form to be ready
	time.Sleep(time.Millisecond * 2000)

	// Create filter
	if err = page.Locator("button[type='submit']").Click(); err != nil {
		return fmt.Errorf("could not click Create button: %v", err)
	}

	// Verify filter was created
	if err = waitForElement(page, fmt.Sprintf("text=%s", filterName), 5000); err != nil {
		return fmt.Errorf("filter was not created successfully: %v", err)
	}
	return nil
}

func testNotifications(page playwright.Page) error {
	// Navigate to notifications settings
	_, err := page.Goto(baseUrl("/settings/notifications"))
	if err != nil {
		return fmt.Errorf("could not navigate to notifications settings: %v", err)
	}

	// Add new notification and wait for slide-over transition
	assertErrorToNilf("could not find and click 'Add new' button: %v", page.Locator("button", playwright.PageLocatorOptions{
		HasText: "Add new",
	}).Click())

	// Wait for slide-over transition to complete
	time.Sleep(time.Millisecond * 700)

	// Wait for and select Discord from dropdown
	assertErrorToNilf("could not wait for notification type dropdown: %v", page.Locator("[id^='react-select'][id$='-input']").WaitFor())
	if err := selectFromDropdown(page, "[id^='react-select'][id$='-input']", "Discord"); err != nil {
		return fmt.Errorf("could not select notification type: %v", err)
	}

	// Wait for Discord form fields to appear
	time.Sleep(time.Millisecond * 500)

	// Fill notification name
	assertErrorToNilf("could not fill name: %v", page.Locator("input#name").Fill("Discord Test"))

	// Fill webhook URL
	assertErrorToNilf("could not fill webhook URL: %v", page.Locator("input#webhook").Fill(discordWebhook))

	// Select an event (using the first event checkbox)
	assertErrorToNilf("could not select event: %v", page.Locator("input[type='checkbox']").First().Click())

	// Wait for form to be ready
	time.Sleep(time.Millisecond * 500)

	// Save notification
	assertErrorToNilf("could not click Save button: %v", page.Locator("button[type='submit']").Click())

	// Verify notification was added
	err = waitForElement(page, "text=Discord", 5000)
	if err != nil {
		return fmt.Errorf("notification was not added successfully: %v", err)
	}
	return nil
}

func testIRCSettings(page playwright.Page) error {
	// Navigate to IRC settings
	_, err := page.Goto(baseUrl("/settings/irc"))
	if err != nil {
		return fmt.Errorf("could not navigate to IRC settings: %v", err)
	}

	// Add new IRC network
	assertErrorToNilf("could not find and click 'Add new network' button: %v", page.Locator("button", playwright.PageLocatorOptions{
		HasText: "Add new network",
	}).Click())

	// Wait for form to be ready
	time.Sleep(time.Millisecond * 1000)

	// Fill IRC settings
	assertErrorToNilf("could not fill name: %v", page.Locator("input#name").Fill("Test Network"))
	assertErrorToNilf("could not fill server: %v", page.Locator("input#server").Fill(ircServer))
	assertErrorToNilf("could not fill nick: %v", page.Locator("input#nick").Fill(ircNick))
	assertErrorToNilf("could not fill auth account: %v", page.Locator("input#auth\\.account").Fill(ircNick))
	assertErrorToNilf("could not fill auth password: %v", page.Locator("input#auth\\.password").Fill("testpass"))

	// Uncheck enabled switch (it's true by default)
	assertErrorToNilf("could not click enabled switch: %v", page.Locator("button[role='switch'][aria-checked='true']").Click())

	// Wait for switch state to be updated
	time.Sleep(time.Millisecond * 500)

	// Create IRC network
	assertErrorToNilf("could not click Create button: %v", page.Locator("button", playwright.PageLocatorOptions{
		HasText: "Create",
	}).Click())

	// Verify IRC network was added
	err = waitForElement(page, fmt.Sprintf("text=%s", ircServer), 5000)
	if err != nil {
		return fmt.Errorf("IRC network was not added successfully: %v", err)
	}
	return nil
}

func testAPISettings(page playwright.Page) error {
	// Navigate to API settings
	_, err := page.Goto(baseUrl("/settings/api"))
	if err != nil {
		return fmt.Errorf("could not navigate to API settings: %v", err)
	}

	// Add new API key and wait for slide-over transition
	assertErrorToNilf("could not find and click 'Add new' button: %v", page.Locator("button", playwright.PageLocatorOptions{
		HasText: "Add new",
	}).Click())

	// Wait for slide-over transition to complete
	time.Sleep(time.Millisecond * 700)

	// Fill API key name
	assertErrorToNilf("could not fill API key name: %v", page.Locator("input#name").Fill(apiKey))

	// Create API key (using specific class to target the correct button)
	assertErrorToNilf("could not click Create button: %v", page.Locator("button.bg-blue-600[type='submit']").Click())

	// Verify API key was added
	err = waitForElement(page, fmt.Sprintf("text=%s", apiKey), 5000)
	if err != nil {
		return fmt.Errorf("API key was not added successfully: %v", err)
	}
	return nil
}

func testApplicationSettings(page playwright.Page) error {
	// Navigate to logs settings
	_, err := page.Goto(baseUrl("/settings/logs"))
	if err != nil {
		return fmt.Errorf("could not navigate to logs settings: %v", err)
	}

	// Wait for and change log level
	assertErrorToNilf("could not wait for log level dropdown: %v", page.Locator("[id^='react-select'][id$='-input']").WaitFor())
	if err := selectFromDropdown(page, "[id^='react-select'][id$='-input']", logLevel); err != nil {
		return fmt.Errorf("could not select log level: %v", err)
	}

	// Wait for settings to be saved automatically
	err = waitForElement(page, "text=Config successfully updated!", 5000)
	if err != nil {
		return fmt.Errorf("log settings were not saved successfully: %v", err)
	}
	return nil
}

// Helper function to add a download client and return any error
func addDownloadClient(page playwright.Page, name, host string) error {
	// Add new client and wait for slide-over animation
	if err := clickButton(page, "Add new client"); err != nil {
		return err
	}

	// Wait for slide-over animation to complete
	time.Sleep(time.Millisecond * 700)

	// Wait for form to be ready and fill name
	assertErrorToNilf("could not wait for name field: %v", page.Locator("input#name").WaitFor())
	if err := fillInput(page, "input#name", name); err != nil {
		return err
	}

	// Wait for and fill host field
	assertErrorToNilf("could not wait for host field: %v", page.Locator("input#host").WaitFor())
	if err := fillInput(page, "input#host", host); err != nil {
		return err
	}

	// Wait for form validation
	time.Sleep(time.Millisecond * 500)

	// Create client
	if err := page.Locator("button[type='submit']").Click(); err != nil {
		return fmt.Errorf("could not click Create button: %v", err)
	}

	// Verify client was added
	err := waitForElement(page, fmt.Sprintf("text=%s", name), 5000)
	if err != nil {
		return fmt.Errorf("download client %s was not added successfully: %v", name, err)
	}

	return nil
}

func testDownloadClients(page playwright.Page) error {
	// Navigate to download clients settings
	_, err := page.Goto(baseUrl("/settings/clients"))
	if err != nil {
		return fmt.Errorf("could not navigate to download clients settings: %v", err)
	}

	// Add qBittorrent client
	if err := addDownloadClient(page, "qbit-test", qbitHost); err != nil {
		return fmt.Errorf("failed to add qBittorrent client: %v", err)
	}

	return nil
}
