---
title: "Resource URIs"
description: "Use ror as a database/sql-style driver so a host program can address ROR organizations as ror:// URIs."
weight: 20
---

`ror` is a command line, but the `ror` Go package is also a
small driver that makes ROR organizations addressable as resource URIs. A host
program registers it the way a program registers a database driver with
`database/sql`, then dereferences `ror://` URIs without knowing
anything about how the ROR API is fetched.

The host that does this today is [ant](https://github.com/tamnd/ant), a single
binary that puts one URI namespace over a family of site tools. The examples
below use `ant`; any program that links the package gets the same behaviour.

## Mounting the driver

A host enables the driver with one blank import, exactly like `import _
"github.com/lib/pq"`:

```go
import _ "github.com/tamnd/ror-cli/ror"
```

The package's `init` registers a domain with the scheme `ror` for the
host `api.ror.org`. The standalone `ror` binary does not change.

## Addressing records

A URI is `scheme://authority/id`. The current resource type:

| URI                            | What it is                                        |
| ------------------------------ | ------------------------------------------------- |
| `ror://org/<ror-id>`           | an organization, keyed by its short ROR ID        |

```bash
ant get ror://org/00f54p054                        # the org record
ant url ror://org/00f54p054                        # the live ror.org URL
ant resolve https://ror.org/00f54p054             # a pasted link, back to its URI
```

As you add resolver operations in `ror/domain.go`, each new `URIType`
becomes another addressable authority here, with no extra wiring. See
[add a command](/guides/adding-a-command/).

## Why this is the same code

The driver and the binary share one definition per operation. A resolver op
answers both `ror org` on the command line and `ant get ror://org/...`
through a host, from the same handler and the same client. There is no second
implementation to keep in step.
