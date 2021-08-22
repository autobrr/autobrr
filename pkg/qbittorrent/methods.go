package qbittorrent

import (
	"encoding/json"
	"errors"
	"io/ioutil"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/rs/zerolog/log"
)

// Login https://github.com/qbittorrent/qBittorrent/wiki/WebUI-API-(qBittorrent-4.1)#authentication
func (c *Client) Login() error {
	credentials := make(map[string]string)
	credentials["username"] = c.settings.Username
	credentials["password"] = c.settings.Password

	resp, err := c.post("auth/login", credentials)
	if err != nil {
		log.Error().Err(err).Msg("login error")
		return err
	} else if resp.StatusCode == http.StatusForbidden {
		log.Error().Err(err).Msg("User's IP is banned for too many failed login attempts")
		return err

	} else if resp.StatusCode != http.StatusOK { // check for correct status code
		log.Error().Err(err).Msg("login bad status error")
		return err
	}

	defer resp.Body.Close()

	bodyBytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	bodyString := string(bodyBytes)

	// read output
	if bodyString == "Fails." {
		return errors.New("bad credentials")
	}

	// good response == "Ok."

	// place cookies in jar for future requests
	if cookies := resp.Cookies(); len(cookies) > 0 {
		c.setCookies(cookies)
	} else {
		return errors.New("bad credentials")
	}

	return nil
}

func (c *Client) GetTorrents() ([]Torrent, error) {
	var torrents []Torrent

	resp, err := c.get("torrents/info", nil)
	if err != nil {
		log.Error().Err(err).Msg("get torrents error")
		return nil, err
	}

	defer resp.Body.Close()

	body, readErr := ioutil.ReadAll(resp.Body)
	if readErr != nil {
		log.Error().Err(err).Msg("get torrents read error")
		return nil, readErr
	}

	err = json.Unmarshal(body, &torrents)
	if err != nil {
		log.Error().Err(err).Msg("get torrents unmarshal error")
		return nil, err
	}

	return torrents, nil
}

func (c *Client) GetTorrentsFilter(filter TorrentFilter) ([]Torrent, error) {
	var torrents []Torrent

	v := url.Values{}
	v.Add("filter", string(filter))
	params := v.Encode()

	resp, err := c.get("torrents/info?"+params, nil)
	if err != nil {
		log.Error().Err(err).Msgf("get filtered torrents error: %v", filter)
		return nil, err
	}

	defer resp.Body.Close()

	body, readErr := ioutil.ReadAll(resp.Body)
	if readErr != nil {
		log.Error().Err(err).Msgf("get filtered torrents read error: %v", filter)
		return nil, readErr
	}

	err = json.Unmarshal(body, &torrents)
	if err != nil {
		log.Error().Err(err).Msgf("get filtered torrents unmarshal error: %v", filter)
		return nil, err
	}

	return torrents, nil
}

func (c *Client) GetTorrentsRaw() (string, error) {
	resp, err := c.get("torrents/info", nil)
	if err != nil {
		log.Error().Err(err).Msg("get torrent trackers raw error")
		return "", err
	}

	defer resp.Body.Close()

	data, _ := ioutil.ReadAll(resp.Body)

	return string(data), nil
}

func (c *Client) GetTorrentTrackers(hash string) ([]TorrentTracker, error) {
	var trackers []TorrentTracker

	params := url.Values{}
	params.Add("hash", hash)

	p := params.Encode()

	resp, err := c.get("torrents/trackers?"+p, nil)
	if err != nil {
		log.Error().Err(err).Msgf("get torrent trackers error: %v", hash)
		return nil, err
	}

	defer resp.Body.Close()

	body, readErr := ioutil.ReadAll(resp.Body)
	if readErr != nil {
		log.Error().Err(err).Msgf("get torrent trackers read error: %v", hash)
		return nil, readErr
	}

	err = json.Unmarshal(body, &trackers)
	if err != nil {
		log.Error().Err(err).Msgf("get torrent trackers: %v", hash)
		return nil, err
	}

	return trackers, nil
}

// AddTorrentFromFile add new torrent from torrent file
func (c *Client) AddTorrentFromFile(file string, options map[string]string) error {

	res, err := c.postFile("torrents/add", file, options)
	if err != nil {
		log.Error().Err(err).Msgf("add torrents error: %v", file)
		return err
	} else if res.StatusCode != http.StatusOK {
		log.Error().Err(err).Msgf("add torrents bad status: %v", file)
		return err
	}

	defer res.Body.Close()

	return nil
}

func (c *Client) DeleteTorrents(hashes []string, deleteFiles bool) error {
	v := url.Values{}

	// Add hashes together with | separator
	hv := strings.Join(hashes, "|")
	v.Add("hashes", hv)
	v.Add("deleteFiles", strconv.FormatBool(deleteFiles))

	encodedHashes := v.Encode()

	resp, err := c.get("torrents/delete?"+encodedHashes, nil)
	if err != nil {
		log.Error().Err(err).Msgf("delete torrents error: %v", hashes)
		return err
	} else if resp.StatusCode != http.StatusOK {
		log.Error().Err(err).Msgf("delete torrents bad code: %v", hashes)
		return err
	}

	defer resp.Body.Close()

	return nil
}

func (c *Client) ReAnnounceTorrents(hashes []string) error {
	v := url.Values{}

	// Add hashes together with | separator
	hv := strings.Join(hashes, "|")
	v.Add("hashes", hv)

	encodedHashes := v.Encode()

	resp, err := c.get("torrents/reannounce?"+encodedHashes, nil)
	if err != nil {
		log.Error().Err(err).Msgf("re-announce error: %v", hashes)
		return err
	} else if resp.StatusCode != http.StatusOK {
		log.Error().Err(err).Msgf("re-announce error bad status: %v", hashes)
		return err
	}

	defer resp.Body.Close()

	return nil
}

func (c *Client) GetTransferInfo() (*TransferInfo, error) {
	var info TransferInfo

	resp, err := c.get("transfer/info", nil)
	if err != nil {
		log.Error().Err(err).Msg("get torrents error")
		return nil, err
	}

	defer resp.Body.Close()

	body, readErr := ioutil.ReadAll(resp.Body)
	if readErr != nil {
		log.Error().Err(err).Msg("get torrents read error")
		return nil, readErr
	}

	err = json.Unmarshal(body, &info)
	if err != nil {
		log.Error().Err(err).Msg("get torrents unmarshal error")
		return nil, err
	}

	return &info, nil
}
