# Current State

Last updated: 2026-06-20

## Repository

- App: Degu Desktop
- Module: `degu-desktop`
- Main app: `cmd/degu`
- Importer: `cmd/importsheet`
- Static site: `docs`
- GitHub workflows: `.github/workflows/pages.yml`, `.github/workflows/release.yml`
- Windows artifacts: x64 (`amd64`) and x86 (`386`)

## Current Architecture

- The app is a Go + Win32 desktop pet overlay.
- Runtime pet state currently lives mostly in `cmd/degu/main_windows.go`.
- Coat variants are represented by a global `variants` list.
- Runtime sprites are normalized to 96x64 frames.
- Existing animations are `idle`, `walk`, `scurry`, `nibble`, `hop`, `turn`, `eat`, `dig`, `stand`, and `groomface`.
- The app currently has degu-only sprite assets, with coat variants handled as degu coats.

## Current Features

- Transparent always-on-top taskbar overlay.
- Tray menu and Japanese/English settings window for optional per-pet names, coat, speed, count, mode, typing wheel, update checks, settings, and exit.
- Keyboard reaction and random stroll modes.
- Typing wheel behavior.
- Foraging props, eating, digging, and carrying behavior.
- Grooming/social pause behavior.
- Optional cursor-hover name labels above visible degus.
- Startup tray notification and GitHub Release based update check/install flow.
- GitHub Pages and GitHub Release workflows, with Pages download metadata for version, update date, and commit.

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

The canonical ImageGen motion set is `wild_agouti`. The importer normalizes 56 source frames and expands them into all configured coat variants.

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

## Known Problems

- Species and coat are not yet separate concepts.
- Degu, chinchilla, and macaroni mouse are not yet independently modeled.
- Some generated degu action frames have inconsistent orientation, so runtime frame selection avoids known unstable walk-cycle frames.
- The current settings window is implemented with animal/motion tabs and Japanese/English labels, but it is still native Win32 rather than a fully custom-rendered UI.
- The current website describes Degu Desktop but does not yet present a multi-species roadmap.
- Future art-style selection is not implemented yet. The user likes the more natural illustrated degu-sheet style from the latest attached reference, so style profiles should be modeled separately from coat variants when this is added.

## Current In-Progress Diff

At the start of this baseline pass, `cmd/degu/main_windows.go` had local uncommitted changes for:

- Natural turn-state motion.
- Bidirectional walking.
- A Win32 settings window.

The active turn implementation uses eight ImageGen-generated frames at runtime frame indices 32-39. Right-to-left uses the source sequence directly; left-to-right mirrors the same sequence.

The user later explicitly requested release publication, so the current finish line includes commit, push, tag, GitHub Release, and Pages verification.

## Codex Config Application

`UDteach/codex_config` was cloned under `.codex/external/codex_config` for reference only. Safe workflow guidance from `codex/AGENTS.md` and relevant skills was applied to this repository's local `AGENTS.md`.

The global restore script was not run because `codex/config.toml` contains machine-specific macOS paths. Restoring it into this Windows environment could break the active Codex setup.
