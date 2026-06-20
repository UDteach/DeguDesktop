# Degu Desktop

Desktop pet app written in Go. On Windows, pixel-art degus walk above the taskbar with the full Degu Desktop behavior set. On macOS, the initial port runs as a menu-bar app and lets degus wander along the bottom edge above the Dock.

Repository: <https://github.com/UDteach/DeguDesktop>

## Features

Windows:

- Transparent always-on-top pet layer above the Windows taskbar
- Tray menu and Japanese/English settings window for names, speed, count, coat color, wheel motion, mode, updates, and exit
- Modes: keyboard reaction / random stroll
- Degu count: 1, 2, 3, 5, or 10
- Optional per-pet names can be enabled in settings; when enabled, names appear above a degu while the cursor hovers over it
- Startup tray notification and GitHub Release based update checking from the tray menu
- Social behavior: nearby degus can walk together and pause for grooming
- Foraging behavior: hay, twigs, and low-key seed-like bits appear near the taskbar for sniffing, eating, digging, gnawing, and carrying
- Wheel motion: in keyboard mode, a degu runs inside the wheel only while you are typing, then scurries away
- Turn motion: eight ImageGen frames smooth direction changes instead of instant sprite flipping
- Coat variants: wild agouti, black, blue-gray, gray, white/cream, sand/champagne, chocolate, black pied, agouti pied, blue pied, and cream pied
- Pied coats use ImageGen coat-guide images for irregular white patch placement, not simple recolors or oval procedural masks
- ImageGen frame PNGs are the art source; no local generated-art fallback

macOS initial port:

- Transparent always-on-top bottom overlay above the Dock area
- Menu-bar degu icon with Quit
- Multiple degus wandering along the bottom edge
- Keyboard reaction through macOS event monitoring when system permissions allow it

## ImageGen Asset Source

Preferred intake is one ImageGen PNG per runtime frame for the canonical wild agouti motion set:

```text
assets/source/frames/wild_agouti/<frame>_<action>_<step>.png
```

Runtime frame contract:

- 56 files for the canonical `wild_agouti` motion set
- actions: idle, walk, scurry, nibble, hop, turn, eat, dig, stand, groomface
- each file contains one complete degu, not a grid
- the importer normalizes every frame into a fixed 96x64 runtime canvas
- the importer expands the canonical motion set into all coat variants

Pied coat guides are also ImageGen sources:

```text
assets/source/coat-guides/<coat_id>.png
```

The importer normalizes each guide and transfers its irregular white patch map across every runtime motion frame for the matching pied coat.

The tray/app icon is also an ImageGen source:

```text
assets/source/imagegen-icon.png
```

The importer normalizes it into `assets/tray.ico`.

Fallback ImageGen action sheets are also supported:

```text
assets/source/imagegen-idle.png
assets/source/imagegen-walk.png
assets/source/imagegen-scurry.png
assets/source/imagegen-nibble.png
assets/source/imagegen-hop.png
assets/source/imagegen-turn.png
assets/source/imagegen-eat.png
assets/source/imagegen-dig.png
assets/source/imagegen-stand.png
assets/source/imagegen-groomface.png
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

Build a macOS app bundle:

```bash
GOARCH=arm64 VERSION=dev ./scripts/build_macos.sh
```

The macOS app runs as a menu-bar app and places a click-through transparent pet layer at the bottom of the current screen, above the Dock area. Global keyboard reaction can require macOS input monitoring/accessibility permission depending on the user's system settings.

## Release

Push a `v*` tag to build Windows ZIPs and attach them to a GitHub Release. GitHub Pages publishes `docs/`. macOS ZIPs are generated with `scripts/build_macos.sh` and can be attached to the same GitHub Release.

Release assets use:

- `DeguDesktop-windows-amd64.zip`
- `DeguDesktop-windows-386.zip`
- `DeguDesktop-macos-arm64.zip`
- `DeguDesktop-macos-amd64.zip`

The Windows app checks `UDteach/DeguDesktop` Releases for the latest matching architecture zip; when a newer release is available, the tray menu can download the zip, stage a temporary updater script, exit, replace `DeguDesktop.exe`, and restart. The macOS app is currently packaged as an ad-hoc-signed app bundle; Developer ID signing and notarization are still separate release-operator steps.

The GitHub Pages workflow also stamps the download area with the Pages build version, JST update date, and short commit ID.

## Cloudflare Pages

`wrangler.jsonc` sets `docs/` as the Pages output directory. For a Git-connected Cloudflare Pages project, connect this repository, use `main` as the production branch, leave the build command blank, and set the output directory to `docs`.

Existing Direct Upload Pages projects cannot be converted to Git integration later, so a Direct Upload project such as `kdevelopk.pages.dev` needs a new Git-connected Pages project if automatic GitHub deploys are required.
