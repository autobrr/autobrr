package qbittorrent

import (
	"bytes"
	"crypto/tls"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/rs/zerolog/log"
	"golang.org/x/net/publicsuffix"
)

var (
	backoffSchedule = []time.Duration{
		5 * time.Second,
		10 * time.Second,
		20 * time.Second,
	}
	timeout = 20 * time.Second
)

type Client struct {
	Name     string
	settings Settings
	http     *http.Client
}

type Settings struct {
	Hostname      string
	Port          uint
	Username      string
	Password      string
	TLS           bool
	TLSSkipVerify bool
	protocol      string
}

func NewClient(s Settings) *Client {
	jarOptions := &cookiejar.Options{PublicSuffixList: publicsuffix.List}
	//store cookies in jar
	jar, err := cookiejar.New(jarOptions)
	if err != nil {
		log.Error().Err(err).Msg("new client cookie error")
	}
	httpClient := &http.Client{
		Timeout: timeout,
		Jar:     jar,
	}

	c := &Client{
		settings: s,
		http:     httpClient,
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
	reqUrl := fmt.Sprintf("%v://%v:%v/api/v2/%v", c.settings.protocol, c.settings.Hostname, c.settings.Port, endpoint)

	var err error
	var resp *http.Response

	req, err := http.NewRequest("GET", reqUrl, nil)
	if err != nil {
		log.Error().Err(err).Msgf("GET: error %v", reqUrl)
		return nil, err
	}

	// try request and if fail run 3 retries
	for i, backoff := range backoffSchedule {
		resp, err = c.http.Do(req)

		// request ok, lets break out of the loop
		if err == nil {
			break
		}

		log.Debug().Msgf("qbit GET failed: retrying attempt %d - %v", i, reqUrl)

		time.Sleep(backoff)
	}

	if err != nil {
		log.Error().Err(err).Msgf("GET: do %v", reqUrl)
		return nil, err
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

	reqUrl := fmt.Sprintf("%v://%v:%v/api/v2/%v", c.settings.protocol, c.settings.Hostname, c.settings.Port, endpoint)
	req, err := http.NewRequest("POST", reqUrl, strings.NewReader(form.Encode()))
	if err != nil {
		log.Error().Err(err).Msgf("POST: req %v", reqUrl)
		return nil, err
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

		log.Debug().Msgf("qbit POST failed: retrying attempt %d - %v", i, reqUrl)

		time.Sleep(backoff)
	}

	if err != nil {
		log.Error().Err(err).Msgf("POST: do %v", reqUrl)
		return nil, err
	}

	return resp, nil
}

func (c *Client) postFile(endpoint string, fileName string, opts map[string]string) (*http.Response, error) {
	var err error
	var resp *http.Response

	file, err := os.Open(fileName)
	if err != nil {
		log.Error().Err(err).Msgf("POST file: opening file %v", fileName)
		return nil, err
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
		log.Error().Err(err).Msgf("POST file: initializing file field %v", fileName)
		return nil, err
	}

	// Copy the actual file content to the fields writer
	_, err = io.Copy(fileWriter, file)
	if err != nil {
		log.Error().Err(err).Msgf("POST file: could not copy file to writer %v", fileName)
		return nil, err
	}

	// Populate other fields
	if opts != nil {
		for key, val := range opts {
			fieldWriter, err := multiPartWriter.CreateFormField(key)
			if err != nil {
				log.Error().Err(err).Msgf("POST file: could not add other fields %v", fileName)
				return nil, err
			}

			_, err = fieldWriter.Write([]byte(val))
			if err != nil {
				log.Error().Err(err).Msgf("POST file: could not write field %v", fileName)
				return nil, err
			}
		}
	}

	// Close multipart writer
	multiPartWriter.Close()

	reqUrl := fmt.Sprintf("%v://%v:%v/api/v2/%v", c.settings.protocol, c.settings.Hostname, c.settings.Port, endpoint)
	req, err := http.NewRequest("POST", reqUrl, &requestBody)
	if err != nil {
		log.Error().Err(err).Msgf("POST file: could not create request object %v", fileName)
		return nil, err
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

		log.Debug().Msgf("qbit POST file failed: retrying attempt %d - %v", i, reqUrl)

		time.Sleep(backoff)
	}

	if err != nil {
		log.Error().Err(err).Msgf("POST file: could not perform request %v", fileName)
		return nil, err
	}

	return resp, nil
}

func (c *Client) setCookies(cookies []*http.Cookie) {
	cookieURL, _ := url.Parse(fmt.Sprintf("%v://%v:%v", c.settings.protocol, c.settings.Hostname, c.settings.Port))
	c.http.Jar.SetCookies(cookieURL, cookies)
}
