package gitignoreio

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"strings"
	"time"
)

type GitignoreConfig struct {
	Name     string
	Key      string
	Contents string
	FileName string
}

type GitignoreIo struct {
	baseUrl    string
	httpClient *http.Client
	list       []string
}

func NewClient() *GitignoreIo {
	client := &http.Client{
		Timeout: 5 * time.Second,
	}
	return NewClientWithBaseURL("https://www.toptal.com/developers/gitignore/api", client)
}

func NewClientWithBaseURL(baseUrl string, client *http.Client) *GitignoreIo {
	if client == nil {
		client = &http.Client{Timeout: 5 * time.Second}
	}
	return &GitignoreIo{
		baseUrl:    strings.TrimRight(baseUrl, "/"),
		httpClient: client,
	}
}

func (c *GitignoreIo) List() (resp []string, err error) {
	endpoint := c.baseUrl + "/list"
	res, err := c.do(http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	ignoreList := parseIgnoreList(res.Body)

	if len(ignoreList) > 0 {
		return ignoreList, nil
	}
	return nil, errors.New("unable to parse list from gitignore.io")
}

func (c *GitignoreIo) FetchAll() (resp map[string]GitignoreConfig, err error) {
	endpoint := c.baseUrl + "/list?format=json"
	res, err := c.do(http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	var configs map[string]GitignoreConfig

	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}

	if err = json.Unmarshal(body, &configs); err != nil {
		return nil, err
	}

	return configs, nil
}

func (c *GitignoreIo) FetchConfig(names []string) (resp string, err error) {
	endpoint := c.baseUrl + "/" + strings.Join(names, ",")
	res, err := c.do(http.MethodGet, endpoint, nil)
	if err != nil {
		return "", err
	}
	defer res.Body.Close()

	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return "", err
	}

	return string(body), nil
}

func (c *GitignoreIo) do(method, endpoint string, params map[string]string) (*http.Response, error) {
	req, err := http.NewRequest(method, endpoint, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Add("Content-Type", "application/json")
	q := req.URL.Query()
	for key, val := range params {
		q.Set(key, val)
	}
	req.URL.RawQuery = q.Encode()
	res, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	if res.StatusCode < 200 || res.StatusCode >= 300 {
		defer res.Body.Close()
		body, _ := io.ReadAll(res.Body)
		return nil, fmt.Errorf("gitignore.io %s %s: %d %s", method, endpoint, res.StatusCode, strings.TrimSpace(string(body)))
	}
	return res, nil
}

func parseIgnoreList(buf io.Reader) []string {
	var ignoreList []string

	scanner := bufio.NewScanner(buf)
	for scanner.Scan() {
		line := scanner.Text()
		names := strings.Split(line, ",")
		ignoreList = append(ignoreList, names...)
	}

	return ignoreList
}
