# Iteration Log

## Iteration 1 - Baseline And Motion/UI Stabilization

Date: 2026-06-20

### Target

Establish project rules and baseline state, then finish the in-progress natural turn motion and settings window work without publishing.

### Cause

The app previously avoided left-facing generated frames to prevent abrupt left/right flicker. A better runtime behavior is to support bidirectional movement while inserting a turn state between direction changes. The app also needs a normal settings window instead of only tray menus.

### Planned Files

- `AGENTS.md`
- `docs/development/current-state.md`
- `docs/development/iteration-log.md`
- `docs/development/art-direction.md`
- `cmd/degu/main_windows.go`
- `cmd/degu/motion_windows_test.go`

### Baseline Status

- `AGENTS.md`: missing before this iteration.
- `docs/development/`: missing before this iteration.
- Current Git diff at baseline: local changes in `cmd/degu/main_windows.go`.
- Current asset report: `assets/source/import-report.json` exists.
- GitHub Pages workflow exists and uses `docs`.
- Release workflow exists and triggers on `v*` tags.

### Publication Rule

Do not push, create tags, or publish GitHub Releases during this iteration.

### Codex Config Reference

- Cloned `UDteach/codex_config` to `.codex/external/codex_config`.
- Read `codex/AGENTS.md`.
- Read `workflow-accelerator`, `repo-intake`, `codex-memory-qa`, and `skeptic-review` skill guidance.
- Did not run `scripts/restore_to_local.ps1` because it would overwrite global Codex files and the backed-up `config.toml` includes macOS-specific paths.
- Applied safe guidance locally in `AGENTS.md` and `.codex/tasks/degu-desktop-motion-settings.md`.

### Turn Frame Pass

- Replaced the runtime-only squash/flip turn workaround with real ImageGen turn frames.
- Added eight turn frames at source paths `assets/source/frames/wild_agouti/32_turn_00.png` through `39_turn_07.png`.
- Expanded the runtime frame contract from 32 to 40 frames in the first pass.
- Kept the source sequence as right-to-left and mirrored it at draw time for left-to-right turns.

## Iteration 2 - Motion Pack, Settings, Coats, And Release

Date: 2026-06-20

### Target

Finish the user-facing release slice: smoother motion, more action frames, modernized settings controls, Japanese/English labels, selectable coat colors, ImageGen-backed pied coats, and GitHub/Pages release publication.

### Asset Expansion

- Expanded the runtime frame contract from 40 to 56 frames.
- Added ImageGen frame sources for eating, digging, standing, and face grooming.
- Added ImageGen forage prop sources for hay, twig, and a subdued seed-like bit.
- Kept every runtime degu frame normalized to a fixed `96x64` canvas.

### Coat Pass

- Added ImageGen coat guides under `assets/source/coat-guides`.
- Used the guides for black pied, agouti pied, blue pied, and cream pied patch placement.
- Removed the saturated-blue look from blue/blue pied and shifted it toward low-saturation slate/greige.
- Fixed cream pied so the patch area reads as visible white/cream pied rather than a low-contrast recolor.
- Added `docs/development/coat-rgb-audit.md` with reference pages, approximate RGB samples, and patch-ratio notes.

### App Pass

- Added animal/motion tabs to the settings window.
- Added Japanese/English UI labels, defaulting to Japanese.
- Added settings controls for degu count, coat color, mode, speed, typing wheel, and natural turns.
- Added runtime states for eating, digging, standing, and face grooming.
- Kept wheel motion tied to keyboard mode typing.

### Publication Rule

The user explicitly requested release publication after the initial no-publish baseline. This iteration may commit, push, tag, and verify GitHub Pages / GitHub Release.
