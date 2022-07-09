package qbittorrent

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"net/http/httputil"
	"strconv"
	"strings"

	"github.com/autobrr/autobrr/pkg/errors"
)

// Login https://github.com/qbittorrent/qBittorrent/wiki/WebUI-API-(qBittorrent-4.1)#authentication
func (c *client) Login() error {
	opts := map[string]string{
		"username": c.settings.Username,
		"password": c.settings.Password,
	}

	resp, err := c.postBasic("auth/login", opts)
	if err != nil {
		return errors.Wrap(err, "login error")
	} else if resp.StatusCode == http.StatusForbidden {
		return errors.New("User's IP is banned for too many failed login attempts")

	} else if resp.StatusCode != http.StatusOK { // check for correct status code
		return errors.New("qbittorrent login bad status %v", resp.StatusCode)
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

	c.log.Printf("logged into client: %v", c.Name)

	return nil
}

func (c *client) GetTorrents() ([]Torrent, error) {

	resp, err := c.get("torrents/info", nil)
	if err != nil {
		return nil, errors.Wrap(err, "get torrents error")
	}

	defer resp.Body.Close()

	body, readErr := ioutil.ReadAll(resp.Body)
	if readErr != nil {
		return nil, errors.Wrap(readErr, "could not read body")
	}

	var torrents []Torrent
	err = json.Unmarshal(body, &torrents)
	if err != nil {
		return nil, errors.Wrap(err, "could not unmarshal body")
	}

	return torrents, nil
}

func (c *client) GetTorrentsFilter(filter TorrentFilter) ([]Torrent, error) {
	opts := map[string]string{
		"filter": string(filter),
	}

	resp, err := c.get("torrents/info", opts)
	if err != nil {
		return nil, errors.Wrap(err, "could not get filtered torrents with filter: %v", filter)
	}

	defer resp.Body.Close()

	body, readErr := ioutil.ReadAll(resp.Body)
	if readErr != nil {
		return nil, errors.Wrap(readErr, "could not read body")
	}

	var torrents []Torrent
	err = json.Unmarshal(body, &torrents)
	if err != nil {
		return nil, errors.Wrap(err, "could not unmarshal body")
	}

	return torrents, nil
}

func (c *client) GetTorrentsActiveDownloads() ([]Torrent, error) {
	var filter = TorrentFilterDownloading

	opts := map[string]string{
		"filter": string(filter),
	}

	resp, err := c.get("torrents/info", opts)
	if err != nil {
		return nil, errors.Wrap(err, "could not get active torrents")
	}

	defer resp.Body.Close()

	body, readErr := ioutil.ReadAll(resp.Body)
	if readErr != nil {
		return nil, errors.Wrap(readErr, "could not read body")
	}

	var torrents []Torrent
	err = json.Unmarshal(body, &torrents)
	if err != nil {
		return nil, errors.Wrap(readErr, "could not unmarshal body")
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

func (c *client) GetTorrentsRaw() (string, error) {
	resp, err := c.get("torrents/info", nil)
	if err != nil {
		return "", errors.Wrap(err, "could not get torrents raw")
	}

	defer resp.Body.Close()

	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", errors.Wrap(err, "could not get read body torrents raw")
	}

	return string(data), nil
}

func (c *client) GetTorrentTrackers(hash string) ([]TorrentTracker, error) {
	opts := map[string]string{
		"hash": hash,
	}

	resp, err := c.get("torrents/trackers", opts)
	if err != nil {
		return nil, errors.Wrap(err, "could not get torrent trackers for hash: %v", hash)
	}

	defer resp.Body.Close()

	dump, err := httputil.DumpResponse(resp, true)
	if err != nil {
		//c.log.Printf("get torrent trackers error dump response: %v\n", string(dump))
		return nil, errors.Wrap(err, "could not dump response for hash: %v", hash)
	}

	c.log.Printf("get torrent trackers response dump: %q", dump)

	if resp.StatusCode == http.StatusNotFound {
		return nil, nil
	} else if resp.StatusCode == http.StatusForbidden {
		return nil, nil
	}

	body, readErr := ioutil.ReadAll(resp.Body)
	if readErr != nil {
		return nil, errors.Wrap(err, "could not read body")
	}

	c.log.Printf("get torrent trackers body: %v\n", string(body))

	var trackers []TorrentTracker
	err = json.Unmarshal(body, &trackers)
	if err != nil {
		return nil, errors.Wrap(err, "could not unmarshal body")
	}

	return trackers, nil
}

// AddTorrentFromFile add new torrent from torrent file
func (c *client) AddTorrentFromFile(file string, options map[string]string) error {

	res, err := c.postFile("torrents/add", file, options)
	if err != nil {
		return errors.Wrap(err, "could not add torrent %v", file)
	} else if res.StatusCode != http.StatusOK {
		return errors.Wrap(err, "could not add torrent %v unexpected status: %v", file, res.StatusCode)
	}

	defer res.Body.Close()

	return nil
}

func (c *client) DeleteTorrents(hashes []string, deleteFiles bool) error {
	// Add hashes together with | separator
	hv := strings.Join(hashes, "|")

	opts := map[string]string{
		"hashes":      hv,
		"deleteFiles": strconv.FormatBool(deleteFiles),
	}

	resp, err := c.get("torrents/delete", opts)
	if err != nil {
		return errors.Wrap(err, "could not delete torrents: %+v", hashes)
	} else if resp.StatusCode != http.StatusOK {
		return errors.Wrap(err, "could not delete torrents %v unexpected status: %v", hashes, resp.StatusCode)
	}

	defer resp.Body.Close()

	return nil
}

func (c *client) ReAnnounceTorrents(hashes []string) error {
	// Add hashes together with | separator
	hv := strings.Join(hashes, "|")
	opts := map[string]string{
		"hashes": hv,
	}

	resp, err := c.get("torrents/reannounce", opts)
	if err != nil {
		return errors.Wrap(err, "could not re-announce torrents: %v", hashes)
	} else if resp.StatusCode != http.StatusOK {
		return errors.Wrap(err, "could not re-announce torrents: %v unexpected status: %v", hashes, resp.StatusCode)
	}

	defer resp.Body.Close()

	return nil
}

func (c *client) GetTransferInfo() (*TransferInfo, error) {
	resp, err := c.get("transfer/info", nil)
	if err != nil {
		return nil, errors.Wrap(err, "could not get transfer info")
	}

	defer resp.Body.Close()

	body, readErr := ioutil.ReadAll(resp.Body)
	if readErr != nil {
		return nil, errors.Wrap(readErr, "could not read body")
	}

	var info TransferInfo
	err = json.Unmarshal(body, &info)
	if err != nil {
		return nil, errors.Wrap(readErr, "could not unmarshal body")
	}

	return &info, nil
}
