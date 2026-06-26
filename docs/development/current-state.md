# Current State

Last updated: 2026-06-26

## Repository

- App: Degu Desktop
- Module: `degu-desktop`
- Main app: `cmd/degu`
- Importer: `cmd/importsheet`
- Static site: `docs`
- GitHub workflows: `.github/workflows/pages.yml`, `.github/workflows/release.yml`
- Windows artifacts: x64 (`amd64`) and x86 (`386`)
- macOS artifacts: Apple Silicon (`arm64`) and Intel (`amd64`)

## Current Architecture

- The Windows app is a Go + Win32 desktop pet overlay.
- The macOS app is a Go + Cocoa menu-bar app with a transparent bottom overlay above the Dock area.
- Runtime pet state currently lives mostly in `cmd/degu/main_windows.go`.
- The macOS port lives in `cmd/degu/main_darwin.go` and `cmd/degu/darwin_cocoa_darwin.m`.
- Coat variants are represented by a global `variants` list.
- Runtime sprites are normalized to 96x64 frames.
- Existing animations are `idle`, `walk`, `scurry`, `nibble`, `hop`, `turn`, `eat`, `dig`, `stand`, `groomface`, and `wheelrun`.
- The app currently has degu-only sprite assets, with coat variants handled as degu coats.

## Current Windows Features

- Transparent always-on-top taskbar overlay.
- Tray menu and Japanese/English settings window with a home overview for temporary hide/show, optional per-pet names, coat, speed, count, mode, typing wheel, single/multi-monitor display span selection, walking range, display position, update checks/installing, settings, and exit.
- Keyboard reaction and random stroll modes.
- Typing wheel behavior; in random stroll mode, a degu can also occasionally choose the wheel as a random action.
- Foraging props, eating, digging, and carrying behavior.
- Grooming/social pause behavior.
- Optional cursor-hover name labels above visible degus.
- Windows overlay position can be switched between the selected display's taskbar work area and the physical screen bottom, with a saved vertical offset for fine tuning.
- Windows display settings include a monitor/span selector and a saved taskbar walking range, so users can constrain movement to a specific horizontal segment on one monitor or across selected multiple monitors.
- Tray count quick actions cover every visible count from 1 to 10.
- Pet height alignment can be switched between natural staggered lanes and a same-baseline row.
- Startup tray notification and GitHub Release based update check/install flow from the tray menu or Updates settings tab.
- GitHub Pages and GitHub Release workflows, with Pages download metadata for version, update date, and commit.

## Current macOS Port

- Transparent always-on-top click-through layer at the bottom of the current screen, above the Dock visible area.
- Menu-bar degu icon with a native settings window and Quit.
- macOS settings window covers visible count 1-10, fixed/per-pet/random coat selection, optional per-pet names, mode, speed, and typing wheel.
- Multiple degus wander left/right along the bottom edge.
- Keyboard reaction is wired through macOS event monitoring when system permissions allow it.
- Click reactions are wired through global mouse monitoring while preserving the click-through overlay.
- Optional cursor-hover name labels appear above visible degus.
- macOS settings are persisted under the user's Application Support config directory.
- Default macOS builds support macOS 12 Monterey or later.
- Optional Big Sur compatibility ZIPs target macOS 11 using Go 1.24.11 and `MACOS_MIN_VERSION=11.0`; real macOS 11 smoke testing is still outstanding.
- Local packaging uses `scripts/build_macos.sh` to create ad-hoc-signed `DeguDesktop.app` ZIPs.
- Local release packaging creates `DeguDesktop-macos-arm64.zip` and `DeguDesktop-macos-amd64.zip`; compatibility packaging can also create `DeguDesktop-macos-big-sur-arm64.zip` and `DeguDesktop-macos-big-sur-amd64.zip`. Those ZIPs can be attached to GitHub Releases manually.

Known macOS gaps:

- No macOS foraging behavior, update installer, or notarization automation yet.
- Developer ID signing and Apple notarization remain manual release-operator steps.

## Current Asset Format

Preferred source:

```text
assets/source/frames/wild_agouti/<frame>_<action>_<step>.png
```

Runtime output:

```text
assets/sprites/degu_<coat_id>.png
```

The current format does not yet separate animal species from coat.

The canonical ImageGen motion set is `wild_agouti`. The importer normalizes 62 source frames and expands them into all configured coat variants.

Pied variants additionally use ImageGen coat guides:

```text
assets/source/coat-guides/<coat_id>.png
```

Those guides provide irregular light patch placement for black pied, agouti pied, blue pied, and cream pied. They are not procedural oval masks.

The tray/app icon uses an ImageGen source:

```text
assets/source/imagegen-icon.png
```

The importer normalizes it into `assets/tray.ico`.

The typing wheel uses an ImageGen back-layer source:

```text
assets/source/imagegen-wheel.png
```

The runtime draws the rotating front spokes and hub over that back layer, so the wheel source should not include front spokes or a hub cap. The degu running inside the wheel uses six dedicated one-frame ImageGen `wheelrun` source PNGs instead of reusing the normal walk cycle.

## Known Problems

- Species and coat are not yet separate concepts.
- Degu, chinchilla, and macaroni mouse are not yet independently modeled.
- Some generated degu action frames have inconsistent orientation, so runtime frame selection avoids known unstable walk-cycle frames.
- The current settings window is implemented with home/animal/motion/display/update tabs and Japanese/English labels, but it is still native Win32 rather than a fully custom-rendered UI.
- The current website describes Degu Desktop but does not yet present a multi-species roadmap.
- Future art-style selection is not implemented yet. The user likes the more natural illustrated degu-sheet style from the latest attached reference, so style profiles should be modeled separately from coat variants when this is added.

## Current Release State

`v0.1.12` is the latest published Windows release line. It includes:

- A Windows tray menu action for temporarily hiding and restoring the pet overlay during the current session.
- A non-persisted runtime flag and guards so keyboard, click reaction, and hover name behavior do not run while the overlay is hidden.
- A GitHub Pages version-history section near the download area.
- Windows display settings for a selected multi-monitor span, including moving the span and expanding/shrinking it for 3+ monitor layouts.
- Segment-based pet placement so visible-count changes immediately distribute pets across selected multi-monitor spans.
- Walking-range settings that default to all selected displays when entering multi-monitor span mode.
- Walking-range summaries that describe the selected area in screen terms such as all selected displays, display 1 only, or part of displays 1-2 while keeping left/right fine adjustment available.

The Mac download links currently remain on the existing `v0.1.9` artifacts until a separate macOS release sync is built and uploaded.

## Codex Config Application

`UDteach/codex_config` was cloned under `.codex/external/codex_config` for reference only. Safe workflow guidance from `codex/AGENTS.md` and relevant skills was applied to this repository's local `AGENTS.md`.

The global restore script was not run because `codex/config.toml` contains machine-specific macOS paths. Restoring it into this Windows environment could break the active Codex setup.
