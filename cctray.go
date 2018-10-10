package gocdexporter

import (
	"context"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
)

// ParseCCTray expects the cctray.xml contents as input.
func ParseCCTray(input io.Reader) (*CCTray, error) {
	cc := CCTray{}
	d := xml.NewDecoder(input)
	if err := d.Decode(&cc); err != nil {
		return nil, err
	}
	filtered := CCTray{Projects: make([]CCTrayProject, 0)}
	for _, p := range cc.Projects {
		if len(strings.Split(p.Name, " :: ")) == 3 {
			filtered.Projects = append(filtered.Projects, p)
		}
	}
	return &filtered, nil
}

// RequestCCTray performs basic auth against provided URL and calls ParseCCTray.
func RequestCCTray(ctx context.Context, url, user, pass string) (*CCTray, error) {
	req, err := http.NewRequest(
		"GET", fmt.Sprintf("%s/go/%s", url, "cctray.xml"), nil)
	if err != nil {
		return nil, err
	}
	req = req.WithContext(ctx)
	req.SetBasicAuth(user, pass)
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	return ParseCCTray(res.Body)
}

// CCTrayCache keeps a cache of CCTray.
type CCTrayCache struct {
	user string
	pass string
	url  string
	cc   *CCTray
}

func NewCCTrayCache(url, user, pass string) *CCTrayCache {
	return &CCTrayCache{
		url:  url,
		user: user,
		pass: pass,
	}
}

// Get cached CCTray.
func (cache *CCTrayCache) Get(ctx context.Context) (*CCTray, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	if cache.cc == nil {
		return nil, errors.New("cache not updated")
	}
	return cache.cc, nil
}

// Update the CCTray cache.
func (cache *CCTrayCache) Update(ctx context.Context) error {
	var err error
	cache.cc, err = RequestCCTray(ctx, cache.url, cache.user, cache.pass)
	return err
}

type CCTray struct {
	Projects []CCTrayProject `xml:"Project"`
}

type CCTrayProject struct {
	Name     string `xml:"name,attr"`
	Activity string `xml:"activity,attr"`
	URL      string `xml:"webUrl,attr"`

	urlParts []string
}

// Pipeline extracted from name attribute.
func (p CCTrayProject) Pipeline() string {
	parts := strings.Split(p.Name, " :: ")

	if len(parts) > 0 {
		return parts[0]
	}
	return ""
}

// Stage extracted from name attribute.
func (p CCTrayProject) Stage() string {
	parts := strings.Split(p.Name, " :: ")

	if len(parts) > 1 {
		return parts[1]
	}
	return ""
}

// Instance number parsed from webURL attribute. Returns -1 on errors.
func (p CCTrayProject) Instance() int64 {
	parts := strings.Split(p.URL, "/")
	if len(parts) > 8 {
		n, err := strconv.ParseInt(parts[8], 10, 64)
		if err != nil {
			return -1
		}
		return n
	}
	return -1
}
