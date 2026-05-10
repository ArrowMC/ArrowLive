package zlmedia

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type Client struct {
	base   string
	secret string
	http   *http.Client
}

func New(base, secret string) *Client {
	return &Client{
		base:   strings.TrimRight(base, "/"),
		secret: secret,
		http:   &http.Client{Timeout: 5 * time.Second},
	}
}

type baseResp struct {
	Code int             `json:"code"`
	Msg  string          `json:"msg"`
	Data json.RawMessage `json:"data"`
}

func (c *Client) get(path string, q url.Values) (*baseResp, error) {
	if q == nil {
		q = url.Values{}
	}
	q.Set("secret", c.secret)
	u := c.base + path + "?" + q.Encode()
	resp, err := c.http.Get(u)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("zlm http %d: %s", resp.StatusCode, string(body))
	}
	var br baseResp
	if err := json.Unmarshal(body, &br); err != nil {
		return nil, fmt.Errorf("zlm decode: %w (body=%s)", err, string(body))
	}
	return &br, nil
}

type MediaItem struct {
	App              string `json:"app"`
	Stream           string `json:"stream"`
	Schema           string `json:"schema"`
	Vhost            string `json:"vhost"`
	OriginType       int    `json:"originType"`
	OriginTypeStr    string `json:"originTypeStr"`
	TotalReaderCount int    `json:"totalReaderCount"`
	AliveSecond      int    `json:"aliveSecond"`
	CreateStamp      int64  `json:"createStamp"`
}

func (c *Client) GetMediaList(app, schema string) ([]MediaItem, error) {
	q := url.Values{}
	if app != "" {
		q.Set("app", app)
	}
	if schema != "" {
		q.Set("schema", schema)
	}
	br, err := c.get("/index/api/getMediaList", q)
	if err != nil {
		return nil, err
	}
	if br.Code != 0 {
		return nil, fmt.Errorf("zlm getMediaList code=%d msg=%s", br.Code, br.Msg)
	}
	if len(br.Data) == 0 {
		return nil, nil
	}
	var items []MediaItem
	if err := json.Unmarshal(br.Data, &items); err != nil {
		return nil, err
	}
	return items, nil
}

func (c *Client) CloseStream(app, stream, vhost string, force bool) error {
	q := url.Values{}
	q.Set("app", app)
	q.Set("stream", stream)
	if vhost != "" {
		q.Set("vhost", vhost)
	}
	if force {
		q.Set("force", "1")
	}
	br, err := c.get("/index/api/close_stream", q)
	if err != nil {
		return err
	}
	if br.Code != 0 {
		return fmt.Errorf("zlm close_stream code=%d msg=%s", br.Code, br.Msg)
	}
	return nil
}

func (c *Client) KickSession(id string) error {
	q := url.Values{}
	q.Set("id", id)
	br, err := c.get("/index/api/kick_session", q)
	if err != nil {
		return err
	}
	if br.Code != 0 {
		return fmt.Errorf("zlm kick_session code=%d msg=%s", br.Code, br.Msg)
	}
	return nil
}

func (c *Client) KickSessions(localPort int, peerIP string) error {
	q := url.Values{}
	if localPort > 0 {
		q.Set("local_port", fmt.Sprintf("%d", localPort))
	}
	if peerIP != "" {
		q.Set("peer_ip", peerIP)
	}
	br, err := c.get("/index/api/kick_sessions", q)
	if err != nil {
		return err
	}
	if br.Code != 0 {
		return fmt.Errorf("zlm kick_sessions code=%d msg=%s", br.Code, br.Msg)
	}
	return nil
}
