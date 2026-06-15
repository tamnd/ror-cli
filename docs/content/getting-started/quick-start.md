---
title: "Quick start"
description: "Fetch your first record with ror."
weight: 30
---

Once `ror` is on your `PATH`, search for organizations or fetch one by ID. No API
key is required.

```bash
ror search "MIT"
ror filter "types:Education"
ror org 00f54p054
```

By default you get an aligned table. Ask for JSON when you want to pipe it:

```bash
$ ror org 00f54p054 -o json
[
  {
    "id": "https://ror.org/00f54p054",
    "name": "Massachusetts Institute of Technology",
    "short_name": "MIT",
    "country": "United States",
    "city": "Cambridge",
    "status": "active",
    "types": ["Education"],
    "established": 1861,
    "website": "https://www.mit.edu"
  }
]
```

## Shape the output

The same flags work on every command:

```bash
ror search "MIT" --fields id,name,country    # keep only these columns
ror search "MIT" --page 2                    # next page of results
ror filter "types:Healthcare" -o jsonl | jq .name
```

`-o` takes `table`, `json`, `jsonl`, `csv`, `tsv`, `url`, or `raw`. Left to
`auto`, it prints a table to a terminal and JSONL into a pipe, so the same
command reads well by hand and parses cleanly downstream. See
[output formats](/reference/output/) for the full contract.

## Serve it instead

The same operations are available over HTTP and to agents over MCP:

```bash
ror serve --addr :7777 &
curl -s 'localhost:7777/v1/org/00f54p054'    # NDJSON, one record per line
ror mcp                                       # MCP over stdio: search, filter, org
```
