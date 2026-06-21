# Degu Desktop

Desktop pet app written in Go. On Windows, pixel-art degus walk above the taskbar with the full Degu Desktop behavior set. On macOS, the app runs as a menu-bar app and lets degus wander along the bottom edge above the Dock.

Repository: <https://github.com/UDteach/DeguDesktop>

## Features

Windows:

- Transparent always-on-top pet layer above the Windows taskbar
- Tray menu and Japanese/English settings window for names, speed, count, coat color, wheel motion, display position, mode, updates, and exit
- Modes: keyboard reaction / random stroll
- Degu count: settings and tray quick actions support 1-10
- Pet height alignment: choose a natural small stagger or a single shared baseline
- Optional per-pet names can be enabled in settings; when enabled, names appear above a degu while the cursor hovers over it
- Startup tray notification and GitHub Release based update checking from the tray menu
- Social behavior: nearby degus can walk together and pause for grooming
- Foraging behavior: hay, twigs, and low-key seed-like bits appear near the taskbar for sniffing, eating, digging, gnawing, and carrying
- Wheel motion: in keyboard mode, a degu runs inside the wheel while you are typing; in random stroll mode, a degu can occasionally choose the wheel as a natural action
- Turn motion: eight ImageGen frames smooth direction changes instead of instant sprite flipping
- Coat variants: wild agouti, black, blue-gray, gray, white/cream, sand/champagne, chocolate, black pied, agouti pied, blue pied, and cream pied
- Pied coats use ImageGen coat-guide images for irregular white patch placement, not simple recolors or oval procedural masks
- ImageGen frame PNGs are the art source; no local generated-art fallback

macOS:

- Transparent always-on-top bottom overlay above the Dock area
- Menu-bar degu icon with a native settings window
- Settings for visible count 1-10, fixed/per-pet/random coat selection, per-pet names, mode, speed, typing wheel, and exit
- Multiple degus wandering along the bottom edge
- Keyboard reaction through macOS event monitoring when system permissions allow it
- Click reactions and optional cursor-hover name labels while preserving the click-through desktop layer
- Supported OS for default macOS builds: macOS 12 Monterey or later, Intel and Apple Silicon
- Optional Big Sur compatibility ZIPs can be built separately for macOS 11, Intel and Apple Silicon

## ImageGen Asset Source

Preferred intake is one ImageGen PNG per runtime frame for the canonical wild agouti motion set:

```text
assets/source/frames/wild_agouti/<frame>_<action>_<step>.png
```

Runtime frame contract:

- 62 files for the canonical `wild_agouti` motion set
- actions: idle, walk, scurry, nibble, hop, turn, eat, dig, stand, groomface, wheelrun
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
assets/source/imagegen-wheelrun.png
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

The macOS app runs as a menu-bar app and places a click-through transparent pet layer at the bottom of the current screen, above the Dock area. The menu-bar icon opens a native settings window for visible count, coat selection, per-pet names, mode, speed, typing wheel, and exit. It does not show a Dock icon by default. Default macOS builds target macOS 12 Monterey or later. Global keyboard and mouse reaction can require macOS input monitoring/accessibility permission depending on the user's system settings.

Build macOS 11 Big Sur compatibility ZIPs with Go 1.24:

```bash
GOTOOLCHAIN=local GO_CMD=/path/to/go1.24.11/bin/go GOARCH=amd64 VERSION=v0.1.7-big-sur MACOS_MIN_VERSION=11.0 MACOS_COMPAT_LABEL=big-sur ./scripts/build_macos.sh
GOTOOLCHAIN=local GO_CMD=/path/to/go1.24.11/bin/go GOARCH=arm64 VERSION=v0.1.7-big-sur MACOS_MIN_VERSION=11.0 MACOS_COMPAT_LABEL=big-sur ./scripts/build_macos.sh
```

These commands create `DeguDesktop-macos-big-sur-amd64.zip` for Intel Macs and `DeguDesktop-macos-big-sur-arm64.zip` for Apple Silicon Macs. Big Sur support depends on the Go 1.24 compatibility toolchain and should be smoke-tested on a real macOS 11 machine before publishing it as a fully verified release asset.

## Release

Push a `v*` tag to build Windows ZIPs and attach them to a GitHub Release. GitHub Pages publishes `docs/`. macOS ZIPs are generated with `scripts/build_macos.sh` and can be attached to the same GitHub Release.

Release assets use:

- `DeguDesktop-windows-amd64.zip`
- `DeguDesktop-windows-386.zip`
- `DeguDesktop-macos-arm64.zip`
- `DeguDesktop-macos-amd64.zip`
- `DeguDesktop-macos-big-sur-arm64.zip`
- `DeguDesktop-macos-big-sur-amd64.zip`

The Windows app checks `UDteach/DeguDesktop` Releases for the latest matching architecture zip; when a newer release is available, the tray menu can download the zip, stage a temporary updater script, exit, replace `DeguDesktop.exe`, and restart. The macOS app is currently packaged as an ad-hoc-signed app bundle. Default ZIPs target macOS 12 Monterey or later, and optional Big Sur ZIPs target macOS 11 with Go 1.24. Developer ID signing and notarization are still separate release-operator steps.

The GitHub Pages workflow also stamps the download area with the Pages build version, JST update date, and short commit ID.

## Cloudflare Pages

`wrangler.jsonc` sets `docs/` as the Pages output directory. For a Git-connected Cloudflare Pages project, connect this repository, use `main` as the production branch, leave the build command blank, and set the output directory to `docs`.

Existing Direct Upload Pages projects cannot be converted to Git integration later, so a Direct Upload project such as `kdevelopk.pages.dev` needs a new Git-connected Pages project if automatic GitHub deploys are required.
