package ror

import (
	"testing"

	"github.com/tamnd/any-cli/kit"
)

// These tests are offline: they exercise the URI driver's pure string functions
// and the host wiring, which need no network.

func TestDomainInfo(t *testing.T) {
	info := Domain{}.Info()
	if info.Scheme != "ror" {
		t.Errorf("Scheme = %q, want ror", info.Scheme)
	}
	if len(info.Hosts) == 0 || info.Hosts[0] != Host {
		t.Errorf("Hosts = %v, want [%s]", info.Hosts, Host)
	}
	if info.Identity.Binary != "ror" {
		t.Errorf("Identity.Binary = %q, want ror", info.Identity.Binary)
	}
}

func TestClassify(t *testing.T) {
	cases := []struct {
		in, typ, id string
	}{
		{"00f54p054", "org", "00f54p054"},
		{"https://ror.org/00f54p054", "org", "00f54p054"},
		{"http://ror.org/00f54p054", "org", "00f54p054"},
		{"/00f54p054/", "org", "00f54p054"},
	}
	for _, tc := range cases {
		typ, id, err := Domain{}.Classify(tc.in)
		if err != nil || typ != tc.typ || id != tc.id {
			t.Errorf("Classify(%q) = (%q, %q, %v), want (%q, %q, nil)",
				tc.in, typ, id, err, tc.typ, tc.id)
		}
	}
}

func TestClassifyEmpty(t *testing.T) {
	_, _, err := Domain{}.Classify("")
	if err == nil {
		t.Error("Classify(\"\") should return error")
	}
}

func TestLocate(t *testing.T) {
	got, err := Domain{}.Locate("org", "00f54p054")
	want := "https://ror.org/00f54p054"
	if err != nil || got != want {
		t.Errorf("Locate = (%q, %v), want (%q, nil)", got, err, want)
	}
}

func TestLocateUnknownType(t *testing.T) {
	_, err := Domain{}.Locate("page", "00f54p054")
	if err == nil {
		t.Error("Locate with unknown type should return error")
	}
}

// TestHostWiring mounts the driver in a kit Host and checks the round trip.
func TestHostWiring(t *testing.T) {
	h, err := kit.Open()
	if err != nil {
		t.Fatal(err)
	}

	o := &Org{
		ID:   "https://ror.org/00f54p054",
		Name: "MIT",
	}
	u, err := h.Mint(o)
	if err != nil {
		t.Fatalf("Mint: %v", err)
	}
	// The id field is the full ROR URL; stripRORPrefix is applied in Classify.
	_ = u

	got, err := h.ResolveOn("ror", "00f54p054")
	if err != nil {
		t.Fatalf("ResolveOn: %v", err)
	}
	if got.String() != "ror://org/00f54p054" {
		t.Errorf("ResolveOn = %q, want ror://org/00f54p054", got.String())
	}
}
