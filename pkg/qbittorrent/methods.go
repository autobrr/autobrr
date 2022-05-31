package qbittorrent

import (
	"encoding/json"
	"errors"
	"io/ioutil"
	"net/http"
	"net/http/httputil"
	"strconv"
	"strings"

	"github.com/rs/zerolog/log"
)

// Login https://github.com/qbittorrent/qBittorrent/wiki/WebUI-API-(qBittorrent-4.1)#authentication
func (c *Client) Login() error {
	opts := map[string]string{
		"username": c.settings.Username,
		"password": c.settings.Password,
	}

	resp, err := c.postBasic("auth/login", opts)
	if err != nil {
		log.Error().Err(err).Msg("login error")
		return err
	} else if resp.StatusCode == http.StatusForbidden {
		log.Error().Err(err).Msg("User's IP is banned for too many failed login attempts")
		return err

	} else if resp.StatusCode != http.StatusOK { // check for correct status code
		log.Error().Err(err).Msgf("login bad status %v error", resp.StatusCode)
		return errors.New("qbittorrent login bad status")
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

	var torrents []Torrent
	err = json.Unmarshal(body, &torrents)
	if err != nil {
		log.Error().Err(err).Msg("get torrents unmarshal error")
		return nil, err
	}

	return torrents, nil
}

func (c *Client) GetTorrentsFilter(filter TorrentFilter) ([]Torrent, error) {
	opts := map[string]string{
		"filter": string(filter),
	}

	resp, err := c.get("torrents/info", opts)
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

	var torrents []Torrent
	err = json.Unmarshal(body, &torrents)
	if err != nil {
		log.Error().Err(err).Msgf("get filtered torrents unmarshal error: %v", filter)
		return nil, err
	}

	return torrents, nil
}

func (c *Client) GetTorrentsActiveDownloads() ([]Torrent, error) {
	var filter = TorrentFilterDownloading

	opts := map[string]string{
		"filter": string(filter),
	}

	resp, err := c.get("torrents/info", opts)
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

	var torrents []Torrent
	err = json.Unmarshal(body, &torrents)
	if err != nil {
		log.Error().Err(err).Msgf("get filtered torrents unmarshal error: %v", filter)
		return nil, err
	}

	res := make([]Torrent, 0)
	for _, torrent := range torrents {
		// qbit counts paused torrents as downloading as well by default
		// so only add torrents with state downloading, and not pausedDl, stalledDl etc
		if torrent.State == TorrentStateDownloading || torrent.State == TorrentStateStalledDl {
			res = append(res, torrent)
		}
	}

	return res, nil
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
	opts := map[string]string{
		"hash": hash,
	}

	resp, err := c.get("torrents/trackers", opts)
	if err != nil {
		log.Error().Err(err).Msgf("get torrent trackers error: %v", hash)
		return nil, err
	}

	defer resp.Body.Close()

	dump, err := httputil.DumpResponse(resp, true)
	if err != nil {
		log.Error().Err(err).Msgf("get torrent trackers dump response error: %v", err)
		//return nil, readErr
	}

	log.Trace().Msgf("get torrent trackers response dump: %v", string(dump))

	if resp.StatusCode == http.StatusNotFound {
		//return nil, fmt.Errorf("torrent not found: %v", hash)
		return nil, nil
	} else if resp.StatusCode == http.StatusForbidden {
		//return nil, fmt.Errorf("torrent not found: %v", hash)
		return nil, nil
	}

	body, readErr := ioutil.ReadAll(resp.Body)
	if readErr != nil {
		log.Error().Err(err).Msgf("get torrent trackers read error: %v", hash)
		return nil, readErr
	}

	log.Trace().Msgf("get torrent trackers body: %v", string(body))

	var trackers []TorrentTracker
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
	// Add hashes together with | separator
	hv := strings.Join(hashes, "|")

	opts := map[string]string{
		"hashes":      hv,
		"deleteFiles": strconv.FormatBool(deleteFiles),
	}

	resp, err := c.get("torrents/delete", opts)
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
	// Add hashes together with | separator
	hv := strings.Join(hashes, "|")
	opts := map[string]string{
		"hashes": hv,
	}

	resp, err := c.get("torrents/reannounce", opts)
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

	var info TransferInfo
	err = json.Unmarshal(body, &info)
	if err != nil {
		log.Error().Err(err).Msg("get torrents unmarshal error")
		return nil, err
	}

	return &info, nil
}
