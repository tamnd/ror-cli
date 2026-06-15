package ror

import (
	"context"
	"fmt"
	"strings"

	"github.com/tamnd/any-cli/kit"
	"github.com/tamnd/any-cli/kit/errs"
)

// domain.go exposes ror as a kit Domain: a driver that a multi-domain
// host (ant) enables with a single blank import,
//
//	import _ "github.com/tamnd/ror-cli/ror"
//
// exactly as a database/sql program enables a driver with `import _
// "github.com/lib/pq"`. The init below registers it; the host then dereferences
// ror:// URIs by routing to the operations Register installs. The same
// Domain also builds the standalone ror binary (see cli.NewApp), so the
// binary and a host share one source of truth.
func init() { kit.Register(Domain{}) }

// Domain is the ROR driver. It carries no state; the per-run client is
// built by the factory Register hands kit.
type Domain struct{}

// Info describes the scheme, the hostnames a pasted link is matched against, and
// the identity reused for the binary's help and version.
func (Domain) Info() kit.DomainInfo {
	return kit.DomainInfo{
		Scheme: "ror",
		Hosts:  []string{Host},
		Identity: kit.Identity{
			Binary: "ror",
			Short:  "A command line for the Research Organization Registry.",
			Long: `A command line for the Research Organization Registry (ROR).

ror reads public ROR data over plain HTTPS, shapes it into
clean records, and prints output that pipes into the rest of your tools. No API
key, nothing to run alongside it.`,
			Site: "ror.org",
			Repo: "https://github.com/tamnd/ror-cli",
		},
	}
}

// Register installs the client factory and every operation onto app.
func (Domain) Register(app *kit.App) {
	app.SetClient(newClient)

	// search: query by name.
	kit.Handle(app, kit.OpMeta{
		Name:    "search",
		Group:   "read",
		Summary: "Search organizations by name",
		Args:    []kit.Arg{{Name: "query", Help: "search term"}},
	}, searchOrgs)

	// filter: filter by type, country, etc.
	kit.Handle(app, kit.OpMeta{
		Name:    "filter",
		Group:   "read",
		Summary: "Filter organizations (e.g. types:Education)",
		Args:    []kit.Arg{{Name: "filter", Help: "filter expression, e.g. types:Healthcare"}},
	}, filterOrgs)

	// org: fetch a single organization by ROR ID.
	kit.Handle(app, kit.OpMeta{
		Name:     "org",
		Group:    "read",
		Single:   true,
		Summary:  "Fetch an organization by ROR ID",
		URIType:  "org",
		Resolver: true,
		Args:     []kit.Arg{{Name: "id", Help: "ROR ID, e.g. 00f54p054 or https://ror.org/00f54p054"}},
	}, getOrg)
}

// newClient builds the client from the host-resolved config.
func newClient(_ context.Context, cfg kit.Config) (any, error) {
	c := NewClient()
	if cfg.UserAgent != "" {
		c.UserAgent = cfg.UserAgent
	}
	if cfg.Rate > 0 {
		c.Rate = cfg.Rate
	}
	if cfg.Retries > 0 {
		c.Retries = cfg.Retries
	}
	if cfg.Timeout > 0 {
		c.HTTP.Timeout = cfg.Timeout
	}
	return c, nil
}

// --- inputs ---

type searchInput struct {
	Query  string  `kit:"arg" help:"search term"`
	Page   int     `kit:"flag" help:"page number (1-based)"`
	Client *Client `kit:"inject"`
}

type filterInput struct {
	Filter string  `kit:"arg" help:"filter expression, e.g. types:Healthcare"`
	Page   int     `kit:"flag" help:"page number (1-based)"`
	Client *Client `kit:"inject"`
}

type orgInput struct {
	ID     string  `kit:"arg" help:"ROR ID or URL"`
	Client *Client `kit:"inject"`
}

// --- handlers ---

func searchOrgs(ctx context.Context, in searchInput, emit func(*Org) error) error {
	page := in.Page
	if page < 1 {
		page = 1
	}
	orgs, total, err := in.Client.SearchOrgs(ctx, in.Query, page)
	if err != nil {
		return mapErr(err)
	}
	_ = total
	for i := range orgs {
		if err := emit(&orgs[i]); err != nil {
			return err
		}
	}
	return nil
}

func filterOrgs(ctx context.Context, in filterInput, emit func(*Org) error) error {
	page := in.Page
	if page < 1 {
		page = 1
	}
	orgs, total, err := in.Client.FilterOrgs(ctx, in.Filter, page)
	if err != nil {
		return mapErr(err)
	}
	_ = total
	for i := range orgs {
		if err := emit(&orgs[i]); err != nil {
			return err
		}
	}
	return nil
}

func getOrg(ctx context.Context, in orgInput, emit func(*Org) error) error {
	org, err := in.Client.GetOrg(ctx, in.ID)
	if err != nil {
		return mapErr(err)
	}
	return emit(org)
}

// --- Resolver: the URI-native string functions, pure and network-free ---

// Classify turns any accepted input into the canonical (type, id).
func (Domain) Classify(input string) (uriType, id string, err error) {
	id = stripRORPrefix(input)
	if id == "" {
		return "", "", errs.Usage("unrecognized ROR reference: %q", input)
	}
	return "org", id, nil
}

// Locate is the inverse: the live https URL for a (type, id).
func (Domain) Locate(uriType, id string) (string, error) {
	if uriType != "org" {
		return "", errs.Usage("ror has no resource type %q", uriType)
	}
	id = stripRORPrefix(id)
	return fmt.Sprintf("https://ror.org/%s", strings.Trim(id, "/")), nil
}

// mapErr converts a library error into the kit error kind that carries the right
// exit code.
func mapErr(err error) error {
	return err
}
