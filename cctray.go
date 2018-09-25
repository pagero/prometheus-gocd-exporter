package gocdexporter

import (
	"encoding/xml"
	"io"
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
