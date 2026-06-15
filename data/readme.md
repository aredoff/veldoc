# Veldoc

Minimal file browser for viewing a mounted directory in the browser. Designed for Docker: mount a folder, open the UI, read files and rendered Markdown.

> This file is a **demo** of Markdown rendering in Veldoc.
> It repeats the project [README](../README.md) and adds syntax examples below.

---

## Quick start

```bash
docker run -p 8080:8080 -v ./your-folder:/data ghcr.io/aredoff/veldoc
```

Open [http://localhost:8080](http://localhost:8080)

## Docker Compose

```bash
mkdir -p data
echo "# Hello" > data/readme.md
docker compose up --build
```

## Configuration

| Flag / Env | Default | Description |
|---|---|---|
| `--root` / `VELDOC_ROOT` | `/data` | Directory to serve |
| `--addr` / `VELDOC_ADDR` | `:8080` | Listen address |
| `--auth` / `VELDOC_AUTH` | `none` | `none`, `basic`, `form`, `token` |
| `VELDOC_BASIC_USER` | | Basic auth username |
| `VELDOC_BASIC_PASSWORD` | | Basic auth password |
| `VELDOC_FORM_USER` | | Form auth username |
| `VELDOC_FORM_PASSWORD` | | Form auth password |
| `VELDOC_SESSION_SECRET` | | Form auth session secret |
| `VELDOC_TOKEN` | | Bearer token |
| `--poll-interval` / `VELDOC_POLL_INTERVAL` | `3s` | UI poll interval |
| `--max-file-size` / `VELDOC_MAX_FILE_SIZE` | `2097152` | Max readable file size |

## Auth examples

**Basic auth:**

```bash
docker run -p 8080:8080 -v ./data:/data \
  -e VELDOC_AUTH=basic \
  -e VELDOC_BASIC_USER=admin \
  -e VELDOC_BASIC_PASSWORD=secret \
  veldoc
```

**Form auth:**

```bash
docker run -p 8080:8080 -v ./data:/data \
  -e VELDOC_AUTH=form \
  -e VELDOC_FORM_USER=admin \
  -e VELDOC_FORM_PASSWORD=secret \
  -e VELDOC_SESSION_SECRET=change-me \
  veldoc
```

**Token auth:**

```bash
curl -H "Authorization: Bearer my-token" http://localhost:8080/api/tree
```

## Development

```bash
go run ./cmd/veldoc --root ./data
go test ./...
```

## License

MIT

---

# Markdown syntax demo

## Text formatting

Regular text with *italic*, **bold**, ***bold italic***, ~~strikethrough~~, and `inline code`.

## Lists

Unordered:

- directories and files in the sidebar
- raw and markdown preview
- polling updates every few seconds

Ordered:

1. mount a folder into Docker
2. open Veldoc in the browser
3. select a file from the sidebar

Task list:

- [x] render headings
- [x] render tables
- [ ] edit files in browser

## Links

- Project repository: [github.com/aredoff/veldoc](https://github.com/aredoff/veldoc)
- Go website: [go.dev](https://go.dev)
- Local plain text file: [note.txt](./note.txt)
- README in repository root: [README.md](../README.md)

## Image

![Go gopher](https://www.pngplay.com/wp-content/uploads/9/Gopher-PNG-Photos.png)

## Blockquote

> Veldoc is useful when an AI agent works in a folder
> and you want to inspect files without syncing or downloading them.

## Code blocks

```go
package main

import "fmt"

func main() {
    fmt.Println("Hello from Veldoc")
}
```

```json
{
  "path": "readme.md",
  "auth": "none"
}
```

## Table

| Feature | Status |
|---|---|
| Sidebar tree | yes |
| Markdown preview | yes |
| Form auth | yes |
| File editing | no |

## Horizontal rule

---

End of demo.
