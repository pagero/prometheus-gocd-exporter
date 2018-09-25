package gocdexporter

import (
	"os"
	"testing"
)

func TestCCTrayParse(t *testing.T) {
	f, err := os.Open("fixtures/cctray.xml")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	cc, err := ParseCCTray(f)
	if err != nil {
		t.Fatal(err)
	}
	t.Log(cc)

	if n := len(cc.Projects); n != 3 {
		t.Error("invalid length", n)
	}

	p := cc.Projects[0]
	if n := p.Pipeline(); n != "foobar-release-2018-09-19" {
		t.Error("invalid pipeline name", n)
	}
	if n := p.Stage(); n != "build" {
		t.Error("invalid stage name", n)
	}
	if n := p.Instance(); n != 22 {
		t.Error("invalid instance", n)
	}

}
