package qbittorrent

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httputil"
	"strconv"
	"strings"

	"github.com/autobrr/autobrr/pkg/errors"
)

// Login https://github.com/qbittorrent/qBittorrent/wiki/WebUI-API-(qBittorrent-4.1)#authentication
func (c *Client) Login() error {
	opts := map[string]string{
		"username": c.settings.Username,
		"password": c.settings.Password,
	}

	resp, err := c.postBasic("auth/login", opts)
	if err != nil {
		return errors.Wrap(err, "login error")
	}

	defer resp.Body.Close()

	if resp.StatusCode == http.StatusForbidden {
		return errors.New("User's IP is banned for too many failed login attempts")
	} else if resp.StatusCode != http.StatusOK { // check for correct status code
		return errors.New("qbittorrent login bad status %v", resp.StatusCode)
	}

	bodyBytes, err := io.ReadAll(resp.Body)
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

func (c *Client) GetTorrents(fo TorrentFilterOptions) ([]Torrent, error) {
	opts := map[string]string{
		"filter":  string(fo.Filter),
		"sort":    fo.Sort,
		"reverse": strconv.FormatBool(fo.Reverse),
		"limit":   strconv.Itoa(fo.Limit),
		"offset":  strconv.Itoa(fo.Offset),
	}

	if fo.Category != nil {
		opts["category"] = *fo.Category
	}

	if fo.Tag != nil {
		opts["tag"] = *fo.Tag
	}

	if len(fo.Hashes) > 0 {
		opts["hashes"] = strings.Join(fo.Hashes, "|")
	}

	resp, err := c.get("torrents/info", nil)
	if err != nil {
		return nil, errors.Wrap(err, "get torrents error")
	}

	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, errors.Wrap(err, "could not read body")
	}

	var torrents []Torrent
	if err := json.Unmarshal(body, &torrents); err != nil {
		return nil, errors.Wrap(err, "could not unmarshal body")
	}

	return torrents, nil
}

func (c *Client) GetTorrentsActiveDownloads() ([]Torrent, error) {
	torrents, err := c.GetTorrents(TorrentFilterOptions{Filter: TorrentFilterDownloading})
	if err != nil {
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
		return "", errors.Wrap(err, "could not get torrents raw")
	}

	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", errors.Wrap(err, "could not get read body torrents raw")
	}

	return string(data), nil
}

func (c *Client) GetTorrentTrackers(hash string) ([]TorrentTracker, error) {
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

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, errors.Wrap(err, "could not read body")
	}

	c.log.Printf("get torrent trackers body: %v\n", string(body))

	var trackers []TorrentTracker
	if err := json.Unmarshal(body, &trackers); err != nil {
		return nil, errors.Wrap(err, "could not unmarshal body")
	}

	return trackers, nil
}

// AddTorrentFromFile add new torrent from torrent file
func (c *Client) AddTorrentFromFile(file string, options map[string]string) error {

	res, err := c.postFile("torrents/add", file, options)
	if err != nil {
		return errors.Wrap(err, "could not add torrent %v", file)
	}

	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		return errors.Wrap(err, "could not add torrent %v unexpected status: %v", file, res.StatusCode)
	}

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
		return errors.Wrap(err, "could not delete torrents: %+v", hashes)
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return errors.Wrap(err, "could not delete torrents %v unexpected status: %v", hashes, resp.StatusCode)
	}

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
		return errors.Wrap(err, "could not re-announce torrents: %v", hashes)
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return errors.Wrap(err, "could not re-announce torrents: %v unexpected status: %v", hashes, resp.StatusCode)
	}

	return nil
}

func (c *Client) GetTransferInfo() (*TransferInfo, error) {
	resp, err := c.get("transfer/info", nil)
	if err != nil {
		return nil, errors.Wrap(err, "could not get transfer info")
	}

	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, errors.Wrap(err, "could not read body")
	}

	var info TransferInfo
	if err := json.Unmarshal(body, &info); err != nil {
		return nil, errors.Wrap(err, "could not unmarshal body")
	}

	return &info, nil
}

func (c *Client) Resume(hashes []string) error {
	// Add hashes together with | separator
	hv := strings.Join(hashes, "|")
	opts := map[string]string{
		"hashes": hv,
	}

	resp, err := c.get("torrents/resume", opts)
	if err != nil {
		return errors.Wrap(err, "could not resume torrents: %v", hashes)
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return errors.Wrap(err, "could not resume torrents: %v unexpected status: %v", hashes, resp.StatusCode)
	}

	return nil
}

func (c *Client) SetForceStart(hashes []string, value bool) error {
	// Add hashes together with | separator
	hv := strings.Join(hashes, "|")
	opts := map[string]string{
		"hashes": hv,
		"value":  strconv.FormatBool(value),
	}

	resp, err := c.get("torrents/setForceStart", opts)
	if err != nil {
		return errors.Wrap(err, "could not setForceStart torrents: %v", hashes)
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return errors.Wrap(err, "could not setForceStart torrents: %v unexpected status: %v", hashes, resp.StatusCode)
	}

	return nil
}

func (c *Client) Recheck(hashes []string) error {
	// Add hashes together with | separator
	hv := strings.Join(hashes, "|")
	opts := map[string]string{
		"hashes": hv,
	}

	resp, err := c.get("torrents/recheck", opts)
	if err != nil {
		return errors.Wrap(err, "could not recheck torrents: %v", hashes)
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return errors.Wrap(err, "could not recheck torrents: %v unexpected status: %v", hashes, resp.StatusCode)
	}

	return nil
}

func (c *Client) Pause(hashes []string) error {
	// Add hashes together with | separator
	hv := strings.Join(hashes, "|")
	opts := map[string]string{
		"hashes": hv,
	}

	resp, err := c.get("torrents/pause", opts)
	if err != nil {
		return errors.Wrap(err, "could not pause torrents: %v", hashes)
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return errors.Wrap(err, "could not pause torrents: %v unexpected status: %v", hashes, resp.StatusCode)
	}

	return nil
}

func (c *Client) SetAutoManagement(hashes []string, enable bool) error {
	// Add hashes together with | separator
	hv := strings.Join(hashes, "|")
	opts := map[string]string{
		"hashes": hv,
		"enable": strconv.FormatBool(enable),
	}

	resp, err := c.get("torrents/setAutoManagement", opts)
	if err != nil {
		return errors.Wrap(err, "could not setAutoManagement torrents: %v", hashes)
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return errors.Wrap(err, "could not setAutoManagement torrents: %v unexpected status: %v", hashes, resp.StatusCode)
	}

	return nil
}

func (c *Client) CreateCategory(category string, path string) error {
	opts := map[string]string{
		"category": category,
		"savePath": path,
	}

	resp, err := c.get("torrents/createCategory", opts)
	if err != nil {
		return errors.Wrap(err, "could not createCategory torrents: %v", category)
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return errors.Wrap(err, "could not createCategory torrents: %v unexpected status: %v", category, resp.StatusCode)
	}

	return nil
}

func (c *Client) EditCategory(category string, path string) error {
	opts := map[string]string{
		"category": category,
		"savePath": path,
	}

	resp, err := c.get("torrents/editCategory", opts)
	if err != nil {
		return errors.Wrap(err, "could not editCategory torrents: %v", category)
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return errors.Wrap(err, "could not editCategory torrents: %v unexpected status: %v", category, resp.StatusCode)
	}

	return nil
}

func (c *Client) RemoveCategories(categories []string) error {
	opts := map[string]string{
		"categories": strings.Join(categories, "\n"),
	}

	resp, err := c.get("torrents/removeCategories", opts)
	if err != nil {
		return errors.Wrap(err, "could not removeCategories torrents: %v", opts["categories"])
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return errors.Wrap(err, "could not removeCategories torrents: %v unexpected status: %v", opts["categories"], resp.StatusCode)
	}

	return nil
}

func (c *Client) SetCategory(hashes []string, category string) error {
	// Add hashes together with | separator
	hv := strings.Join(hashes, "|")
	opts := map[string]string{
		"hashes":   hv,
		"category": category,
	}

	resp, err := c.get("torrents/setCategory", opts)
	if err != nil {
		return errors.Wrap(err, "could not setCategory torrents: %v", hashes)
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return errors.Wrap(err, "could not setCategory torrents: %v unexpected status: %v", hashes, resp.StatusCode)
	}

	return nil
}

func (c *Client) GetFilesInformation(hash string) (*TorrentFiles, error) {
	opts := map[string]string{
		"hash": hash,
	}

	resp, err := c.get("torrents/files", opts)
	if err != nil {
		return nil, errors.Wrap(err, "could not get files info")
	}

	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, errors.Wrap(err, "could not read body")
	}

	var info TorrentFiles
	if err := json.Unmarshal(body, &info); err != nil {
		return nil, errors.Wrap(err, "could not unmarshal body")
	}

	return &info, nil
}

func (c *Client) GetCategories() (map[string]Category, error) {
	resp, err := c.get("torrents/categories", nil)
	if err != nil {
		return nil, errors.Wrap(err, "could not get files info")
	}

	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, errors.Wrap(err, "could not read body")
	}

	m := make(map[string]Category)
	if err := json.Unmarshal(body, &m); err != nil {
		return nil, errors.Wrap(err, "could not unmarshal body")
	}

	return m, nil
}

func (c *Client) RenameFile(hash, oldPath, newPath string) error {
	opts := map[string]string{
		"hash":    hash,
		"oldPath": oldPath,
		"newPath": newPath,
	}

	resp, err := c.post("torrents/renameFile", opts)
	if err != nil {
		return errors.Wrap(err, "could not renameFile: %v | old: %v | new: %v", hash, oldPath, newPath)
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return errors.Wrap(err, "could not renameFile: %v | old: %v | new: %v unexpected status: %v", hash, oldPath, newPath, resp.StatusCode)
	}

	return nil
}
