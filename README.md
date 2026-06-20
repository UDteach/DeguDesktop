# Degu Desktop

Windows taskbar pet app written in Go. Pixel-art degus walk, stop, scurry, nibble, hop, and run in a wheel while you type above the taskbar.

Repository: <https://github.com/UDteach/DeguDesktop>

## Features

- Transparent always-on-top pet layer above the Windows taskbar
- Tray menu for coat, speed, count, wheel motion, mode, and exit
- Modes: keyboard reaction / random stroll
- Degu count: 1, 2, 3, or 5
- Social behavior: nearby degus can walk together and pause for grooming
- Foraging behavior: hay, twigs, and seed-like bits appear near the taskbar for sniffing, gnawing, and carrying
- Wheel motion: in keyboard mode, a degu runs inside the wheel only while you are typing, then scurries away
- Coat variants: wild agouti, black, blue-gray, gray, white/cream, sand/champagne, chocolate, and pied variants
- ImageGen frame PNGs are the art source; no local generated-art fallback

## ImageGen Asset Source

Preferred intake is one ImageGen PNG per runtime frame:

```text
assets/source/frames/<coat_id>/<frame>_<action>_<step>.png
```

Runtime frame contract:

- 32 files per coat
- actions: idle, walk, scurry, nibble, hop
- each file contains one complete degu, not a grid
- the importer normalizes every frame into a fixed 96x64 runtime canvas

Fallback ImageGen action sheets are also supported:

```text
assets/source/imagegen-idle.png
assets/source/imagegen-walk.png
assets/source/imagegen-scurry.png
assets/source/imagegen-nibble.png
assets/source/imagegen-hop.png
```

Import and validate:

```powershell
go run ./cmd/importsheet
```

The importer writes `assets/sprites/degu_*.png`, `assets/tray.ico`, `docs/assets/degu-preview.png`, and `assets/source/import-report.json`.

## Development

```powershell
go run ./cmd/importsheet
go run ./cmd/degu
```

Build a GUI binary:

```powershell
go build -ldflags="-H=windowsgui" -o dist/DeguDesktop.exe ./cmd/degu
```

## Release

Push a `v*` tag to build `DeguDesktop-windows-amd64.zip` and attach it to a GitHub Release. GitHub Pages publishes `docs/`.

## Cloudflare Pages

`wrangler.jsonc` sets `docs/` as the Pages output directory. For a Git-connected Cloudflare Pages project, connect this repository, use `main` as the production branch, leave the build command blank, and set the output directory to `docs`.

Existing Direct Upload Pages projects cannot be converted to Git integration later, so a Direct Upload project such as `kdevelopk.pages.dev` needs a new Git-connected Pages project if automatic GitHub deploys are required.
