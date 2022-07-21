package qbittorrent

import (
	"bytes"
	"crypto/tls"
	"fmt"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"os"
	"path"
	"strings"
	"time"

	"golang.org/x/net/publicsuffix"

	"github.com/autobrr/autobrr/pkg/errors"
)

var (
	backoffSchedule = []time.Duration{
		5 * time.Second,
		10 * time.Second,
		20 * time.Second,
	}
	timeout = 60 * time.Second
)

type Client struct {
	Name     string
	settings Settings
	http     *http.Client

	log *log.Logger
}

type Settings struct {
	Name          string
	Hostname      string
	Port          uint
	Username      string
	Password      string
	TLS           bool
	TLSSkipVerify bool
	protocol      string
	BasicAuth     bool
	Basic         Basic
	Log           *log.Logger
}

type Basic struct {
	Username string
	Password string
}

func NewClient(settings Settings) *Client {
	c := &Client{
		settings: settings,
		Name:     settings.Name,
		log:      log.New(io.Discard, "", log.LstdFlags),
	}

	// override logger if we pass one
	if settings.Log != nil {
		c.log = settings.Log
	}

	//store cookies in jar
	jarOptions := &cookiejar.Options{PublicSuffixList: publicsuffix.List}
	jar, err := cookiejar.New(jarOptions)
	if err != nil {
		c.log.Println("new client cookie error")
	}

	c.http = &http.Client{
		Timeout: timeout,
		Jar:     jar,
	}

	c.settings.protocol = "http"
	if c.settings.TLS {
		c.settings.protocol = "https"
	}

	if c.settings.TLSSkipVerify {
		//skip TLS verification
		tr := &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		}

		c.http.Transport = tr
	}

	return c
}

func (c *Client) get(endpoint string, opts map[string]string) (*http.Response, error) {
	var err error
	var resp *http.Response

	reqUrl := buildUrlOpts(c.settings, endpoint, opts)

	req, err := http.NewRequest("GET", reqUrl, nil)
	if err != nil {
		return nil, errors.Wrap(err, "could not build request")
	}

	if c.settings.BasicAuth {
		req.SetBasicAuth(c.settings.Basic.Username, c.settings.Basic.Password)
	}

	// try request and if fail run 3 retries
	for i, backoff := range backoffSchedule {
		resp, err = c.http.Do(req)

		// request ok, lets break out of the loop
		if err == nil {
			break
		}

		c.log.Printf("qbit GET failed: retrying attempt %d - %v\n", i, reqUrl)

		time.Sleep(backoff)
	}

	if err != nil {
		return nil, errors.Wrap(err, "error making get request: %v", reqUrl)
	}

	return resp, nil
}

func (c *Client) post(endpoint string, opts map[string]string) (*http.Response, error) {
	// add optional parameters that the user wants
	form := url.Values{}
	if opts != nil {
		for k, v := range opts {
			form.Add(k, v)
		}
	}

	var err error
	var resp *http.Response

	reqUrl := buildUrl(c.settings, endpoint)

	req, err := http.NewRequest("POST", reqUrl, strings.NewReader(form.Encode()))
	if err != nil {
		return nil, errors.Wrap(err, "could not build request")
	}

	if c.settings.BasicAuth {
		req.SetBasicAuth(c.settings.Basic.Username, c.settings.Basic.Password)
	}

	// add the content-type so qbittorrent knows what to expect
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")

	// try request and if fail run 3 retries
	for i, backoff := range backoffSchedule {
		resp, err = c.http.Do(req)

		// request ok, lets break out of the loop
		if err == nil {
			break
		}

		c.log.Printf("qbit POST failed: retrying attempt %d - %v\n", i, reqUrl)

		time.Sleep(backoff)
	}

	if err != nil {
		return nil, errors.Wrap(err, "error making post request: %v", reqUrl)
	}

	return resp, nil
}

func (c *Client) postBasic(endpoint string, opts map[string]string) (*http.Response, error) {
	// add optional parameters that the user wants
	form := url.Values{}
	if opts != nil {
		for k, v := range opts {
			form.Add(k, v)
		}
	}

	var err error
	var resp *http.Response

	reqUrl := buildUrl(c.settings, endpoint)

	req, err := http.NewRequest("POST", reqUrl, strings.NewReader(form.Encode()))
	if err != nil {
		return nil, errors.Wrap(err, "could not build request")
	}

	if c.settings.BasicAuth {
		req.SetBasicAuth(c.settings.Basic.Username, c.settings.Basic.Password)
	}

	// add the content-type so qbittorrent knows what to expect
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")

	resp, err = c.http.Do(req)
	if err != nil {
		return nil, errors.Wrap(err, "error making post request: %v", reqUrl)
	}

	return resp, nil
}

func (c *Client) postFile(endpoint string, fileName string, opts map[string]string) (*http.Response, error) {
	var err error
	var resp *http.Response

	file, err := os.Open(fileName)
	if err != nil {
		return nil, errors.Wrap(err, "error opening file %v", fileName)
	}
	// Close the file later
	defer file.Close()

	// Buffer to store our request body as bytes
	var requestBody bytes.Buffer

	// Store a multipart writer
	multiPartWriter := multipart.NewWriter(&requestBody)

	// Initialize file field
	fileWriter, err := multiPartWriter.CreateFormFile("torrents", fileName)
	if err != nil {
		return nil, errors.Wrap(err, "error initializing file field %v", fileName)
	}

	// Copy the actual file content to the fields writer
	_, err = io.Copy(fileWriter, file)
	if err != nil {
		return nil, errors.Wrap(err, "error copy file contents to writer %v", fileName)
	}

	// Populate other fields
	if opts != nil {
		for key, val := range opts {
			fieldWriter, err := multiPartWriter.CreateFormField(key)
			if err != nil {
				return nil, errors.Wrap(err, "error creating form field %v with value %v", key, val)
			}

			_, err = fieldWriter.Write([]byte(val))
			if err != nil {
				return nil, errors.Wrap(err, "error writing field %v with value %v", key, val)
			}
		}
	}

	// Close multipart writer
	multiPartWriter.Close()

	reqUrl := buildUrl(c.settings, endpoint)
	req, err := http.NewRequest("POST", reqUrl, &requestBody)
	if err != nil {
		return nil, errors.Wrap(err, "error creating request %v", fileName)
	}

	if c.settings.BasicAuth {
		req.SetBasicAuth(c.settings.Basic.Username, c.settings.Basic.Password)
	}

	// Set correct content type
	req.Header.Set("Content-Type", multiPartWriter.FormDataContentType())

	// try request and if fail run 3 retries
	for i, backoff := range backoffSchedule {
		resp, err = c.http.Do(req)

		// request ok, lets break out of the loop
		if err == nil {
			break
		}

		c.log.Printf("qbit POST file failed: retrying attempt %d - %v\n", i, reqUrl)

		time.Sleep(backoff)
	}

	if err != nil {
		return nil, errors.Wrap(err, "error making post file request %v", fileName)
	}

	return resp, nil
}

func (c *Client) setCookies(cookies []*http.Cookie) {
	cookieURL, _ := url.Parse(buildUrl(c.settings, ""))

	c.http.Jar.SetCookies(cookieURL, cookies)
}

func buildUrl(settings Settings, endpoint string) string {
	// parse url
	u, _ := url.Parse(settings.Hostname)

	// reset Opaque
	u.Opaque = ""

	// set scheme
	scheme := "http"
	if u.Scheme == "http" || u.Scheme == "https" {
		if settings.TLS {
			scheme = "https"
		}
		u.Scheme = scheme
	} else {
		if settings.TLS {
			scheme = "https"
		}
		u.Scheme = scheme
	}

	// if host is empty lets use one from settings
	if u.Host == "" {
		u.Host = settings.Hostname
	}

	// reset Path
	if u.Host == u.Path {
		u.Path = ""
	}

	// handle ports
	if settings.Port > 0 {
		if settings.Port == 80 || settings.Port == 443 {
			// skip for regular http and https
		} else {
			u.Host = fmt.Sprintf("%v:%v", u.Host, settings.Port)
		}
	}

	// join path
	u.Path = path.Join(u.Path, "/api/v2/", endpoint)

	// make into new string and return
	return u.String()
}

func buildUrlOpts(settings Settings, endpoint string, opts map[string]string) string {
	// parse url
	u, _ := url.Parse(settings.Hostname)

	// reset Opaque
	u.Opaque = ""

	// set scheme
	scheme := "http"
	if u.Scheme == "http" || u.Scheme == "https" {
		if settings.TLS {
			scheme = "https"
		}
		u.Scheme = scheme
	} else {
		if settings.TLS {
			scheme = "https"
		}
		u.Scheme = scheme
	}

	// if host is empty lets use one from settings
	if u.Host == "" {
		u.Host = settings.Hostname
	}

	// reset Path
	if u.Host == u.Path {
		u.Path = ""
	}

	// handle ports
	if settings.Port > 0 {
		if settings.Port == 80 || settings.Port == 443 {
			// skip for regular http and https
		} else {
			u.Host = fmt.Sprintf("%v:%v", u.Host, settings.Port)
		}
	}

	// add query params
	q := u.Query()
	for k, v := range opts {
		q.Set(k, v)
	}

	u.RawQuery = q.Encode()

	// join path
	u.Path = path.Join(u.Path, "/api/v2/", endpoint)

	// make into new string and return
	return u.String()
}
