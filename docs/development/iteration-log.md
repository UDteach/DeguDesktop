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

## Iteration 3 - Multi-Coat Settings And Layout Fix

Date: 2026-06-20

### Target

Allow multiple coat colors to appear at the same time, keep random and per-pet selection modes available from the settings UI, and fix the settings-window overlap found when increasing visible pets.

### Cause

The previous runtime used one global coat variant for all pets. The settings window also did not reserve enough client-area height for five per-pet color rows, so the added rows could collide with the footer controls on some Windows frame sizes.

### App Pass

- Added fixed, per-pet selected, and random coat appearance modes.
- Stored each pet's runtime coat variant independently instead of drawing every pet from the global coat.
- Added per-pet color combos for up to five visible degus.
- Kept random mode assigning one coat per pet while preserving ImageGen-backed pied variants.
- Made tray coat selection switch back to fixed mode so choosing a coat has an immediate visible effect even after random mode.
- Fixed the settings window to reserve a 760x560 popup client area for the modern two-column layout.
- Converted mode and speed radio choices to push-like segmented buttons for a cleaner Win32-native UI.

### Verification

- `gofmt -w cmd\degu\main_windows.go cmd\degu\motion_windows_test.go`
- `go test -buildvcs=false ./...`
- `go vet -buildvcs=false ./...`
- `go build -buildvcs=false -ldflags="-H=windowsgui" -o dist\DeguDesktop.exe ./cmd/degu`
- `git diff --check`
- Visual QA screenshots:
  - `.codex/qa/settings-window-final-selected-5.png`
  - `.codex/qa/settings-window-final-random.png`
  - `.codex/qa/settings-window-motion-polished.png`
  - `.codex/qa/taskbar-random-5-colors-polished.png`

### UI Library Note

For a more modern settings surface than Win32 stock controls, the best next step is a settings-only WebView2 or Wails panel. Keep the transparent taskbar overlay in Win32 and avoid replacing the whole app with a large GUI framework.

## Iteration 4 - Modern Native Settings Surface

Date: 2026-06-20

### Target

Make the settings window feel current without taking on a large GUI framework or changing the lightweight Win32 taskbar overlay architecture.

### Cause

The previous settings window still relied on stock comboboxes and radio/check controls. The layout was functional, but the old control chrome made the app feel dated.

### App Pass

- Replaced stock settings buttons with owner-drawn pill controls.
- Replaced comboboxes with custom select fields that open native popup menus.
- Added coat color swatches inside fixed and per-pet color selectors.
- Added Yu Gothic UI-based title/body fonts for cleaner Japanese rendering.
- Kept the settings panel dependency-free: no WebView2, Wails, or large GUI toolkit was introduced.
- Kept the existing tray, overlay, random/keyboard mode, multi-coat settings, and release zip path intact.

### Verification

- `gofmt -w cmd\degu\main_windows.go cmd\degu\motion_windows_test.go`
- `go test -buildvcs=false ./...`
- `go vet -buildvcs=false ./...`
- `go build -buildvcs=false -ldflags="-H=windowsgui" -o dist\DeguDesktop.exe ./cmd/degu`
- `git diff --check`
- Relaunched `dist\DeguDesktop.exe`.
- Opened the fixed-color select popup and closed it with Esc; the process stayed responsive.
- Visual QA screenshots:
  - `.codex/qa/settings-window-modern-selected-5.png`
  - `.codex/qa/settings-window-modern-random.png`
  - `.codex/qa/settings-window-modern-motion.png`
  - `.codex/qa/taskbar-modern-random-5-colors.png`

## Iteration 5 - Settings Panel Redesign

Date: 2026-06-20

### Target

Replace the still-dated dialog-like settings surface with a more current desktop-app settings panel while keeping the Win32 overlay architecture lightweight.

### Research Notes

- Microsoft Windows app design guidance and Fluent 2 layout guidance point toward clear navigation, spacing hierarchy, cards, and consistent control shapes.
- Wails can provide Go desktop apps with modern web frontend templates, but adopting it here would change the application structure and packaging more than this iteration requires.
- The chosen path keeps the app dependency-free and applies the useful layout ideas directly in Win32 custom drawing.

### App Pass

- Replaced the standard titled dialog with a borderless 760x560 settings panel.
- Added a dark left navigation rail and a light content surface.
- Moved section labels, page titles, status text, and notes to custom drawing to avoid native static-control artifacts.
- Kept owner-drawn pill buttons and swatch-backed select fields.
- Added top-right close control and drag hit testing for the borderless window header area.
- Fixed selected-coat row spacing so five per-pet coat selectors fit without overlapping the footer.

### Verification

- `gofmt -w cmd\degu\main_windows.go cmd\degu\motion_windows_test.go`
- `go test -buildvcs=false ./...`
- `go vet -buildvcs=false ./...`
- `go build -buildvcs=false -ldflags="-H=windowsgui" -o dist\DeguDesktop.exe ./cmd/degu`
- Updated `dist\DeguDesktop-windows-amd64.zip` and `docs\download\DeguDesktop-windows-amd64.zip`.
- Relaunched `dist\DeguDesktop.exe`.
- Opened color select popup and closed it with Esc; process stayed responsive.
- Verified footer close and top-right close hide the settings window.
- Visual QA screenshots:
  - `.codex/qa/settings-window-panel-final-selected-5.png`
  - `.codex/qa/settings-window-panel-final-random.png`
  - `.codex/qa/settings-window-panel-final-motion.png`
  - `.codex/qa/settings-window-panel-final-close.png`

## Iteration 6 - Settings Position And Click Reactions

Date: 2026-06-20

### Target

Keep the settings panel at the user's current position when switching the left navigation, and add a lightweight degu click reaction without breaking click-through taskbar behavior.

### Cause

The settings panel was destroyed and recreated on tab/language/state changes, but the next window always used the default startup coordinate. The taskbar overlay also had no pointer feedback even though the user expects the pets to react when clicked.

### App Pass

- Stored the current settings panel screen position before recreating the window.
- Reused the stored position when creating the next settings panel.
- Added a low-level mouse hook alongside the existing keyboard hook.
- Kept the overlay click-through by observing global left-clicks and passing them onward.
- Added per-pet reaction state and expiration.
- Added pixel-drawn speech bubbles with heart, smile, and sparkle icons above clicked degus.
- Added focused tests for pet hit testing and reaction lifecycle.

### Verification

- `gofmt -w cmd\degu\main_windows.go cmd\degu\motion_windows_test.go`
- `go test -buildvcs=false ./...`
- `go vet -buildvcs=false ./...`
- `go build -buildvcs=false -ldflags="-H=windowsgui" -o dist\DeguDesktop.exe ./cmd/degu`
- Updated `dist\DeguDesktop-windows-amd64.zip` and `docs\download\DeguDesktop-windows-amd64.zip`.
- Relaunched `dist\DeguDesktop.exe`.
- Moved settings panel to `(280,160)`, switched to the motion tab, and verified the position stayed `(280,160)`.
- Clicked visible taskbar degus and captured a reaction bubble while the underlying desktop click still passed through.
- Visual QA screenshots:
  - `.codex/qa/click-reaction-before.png`
  - `.codex/qa/click-reaction-after-hookfix.png`

## Iteration 7 - Ten Pet Cap And Two-Column Coat Picker

Date: 2026-06-20

### Target

Raise the visible degu cap from five to ten without letting the per-pet coat settings overflow the modern settings panel.

### App Pass

- Increased `maxPetCount` to 10.
- Added a tray menu count option for 10 visible pets.
- Expanded the default selected coat list to ten entries.
- Changed the per-pet coat picker from one vertical list to a two-column, five-row layout.
- Added a layout test to ensure all ten per-pet coat controls fit inside the selected-coats panel.

### Verification

- `gofmt -w cmd\degu\main_windows.go cmd\degu\motion_windows_test.go`
- `go test -buildvcs=false ./...`
- `go vet -buildvcs=false ./...`
- `go run ./cmd/importsheet`
- `go build -buildvcs=false -ldflags="-H=windowsgui" -o dist\DeguDesktop.exe ./cmd/degu`
- `git diff --check`
- Updated `docs\download\DeguDesktop-windows-amd64.zip`.
- Relaunched `dist\DeguDesktop.exe`.
- Sent settings commands for 10 pets and per-pet coat mode.
- Visual QA screenshot:
  - `.codex/qa/settings-ten-pets-selected.png`

## Iteration 8 - Japanese Pages Copy And White Preview

Date: 2026-06-20

### Target

Make the Degu Desktop GitHub Pages site Japanese-first while keeping an English summary, and replace the dark preview thumbnail with a white desktop/taskbar-style preview.

### Cause

The previous Pages copy was English-first. The old generated thumbnail rendered the transparent degu sprites on a black background, which made the site feel inconsistent and made some details harder to inspect.

### Pages Pass

- Changed `docs/index.html` to `lang="ja"` with Japanese-first navigation, lead copy, feature cards, download note, and button text.
- Kept a compact `English` section for non-Japanese visitors.
- Changed `cmd/importsheet` preview generation so `docs/assets/degu-preview.png` is deterministic, white-background, and regenerated the same way in CI.
- Preserved complete visible pets in the preview so bodies and tails do not crop at the thumbnail edges.

### Verification

- `gofmt -w cmd\degu\main_windows.go cmd\degu\motion_windows_test.go cmd\importsheet\main.go cmd\importsheet\main_test.go`
- `go run ./cmd/importsheet`
- `go test -buildvcs=false ./...`
- `go vet -buildvcs=false ./...`
- `go build -buildvcs=false -ldflags="-H=windowsgui" -o dist\DeguDesktop.exe ./cmd/degu`
- Updated `docs\download\DeguDesktop-windows-amd64.zip`.
- `git diff --check`
- Visual QA screenshots:
  - `.codex/qa/degu-pages-ja-white-desktop.png`
  - `.codex/qa/degu-pages-ja-white-mobile.png`

## Iteration 9 - Updates, Icon, And Pet Names

Date: 2026-06-20

### Target

Add a practical Windows update path, improve tray identity with an ImageGen icon, show startup tray notification, and let users name visible degus from settings.

### App Pass

- Added GitHub Release based update checking against `UDteach/DeguDesktop` latest release.
- Added tray menu entries for update check and install when the matching Windows zip is available.
- Added x64/x86 release artifacts and updater asset selection based on the running Go architecture.
- Added a staged updater flow: download release zip, extract `DeguDesktop.exe`, start a temporary PowerShell updater, exit, replace the running exe, and restart.
- Added startup tray balloon notification.
- Added `main.appVersion` injection in Release and Pages workflows.
- Added optional per-pet names to settings persistence.
- Added a `名前を付ける` toggle, name buttons, and a focused rename dialog for up to ten pets.
- Added a hover name popup above the degu under the cursor only when names are enabled.
- Recorded future art-style selection as a roadmap item; the latest natural illustrated degu-sheet reference is a strong target style, but style profiles are not implemented yet.
- Added an ImageGen source icon at `assets/source/imagegen-icon.png` and regenerated `assets/tray.ico` from it.
- Added tests for version comparison, release asset selection, pet-name persistence, and settings row layout.

### Verification

- `gofmt -w cmd\degu\main_windows.go cmd\degu\motion_windows_test.go cmd\importsheet\main.go cmd\importsheet\main_test.go`
- `go test -buildvcs=false ./...`
- `go vet -buildvcs=false ./...`
- `go run ./cmd/importsheet`
- `go build -buildvcs=false -ldflags="-H=windowsgui" -o dist\DeguDesktop.exe ./cmd/degu`
- Built local x64 and x86 ZIPs:
  - `dist\DeguDesktop-windows-amd64.zip`
  - `dist\DeguDesktop-windows-386.zip`
  - `docs\download\DeguDesktop-windows-amd64.zip`
  - `docs\download\DeguDesktop-windows-386.zip`
- Verified each ZIP contains `DeguDesktop.exe` and `README.md`.
- `git diff --check`
- Visual QA screenshots:
  - `.codex/qa/tray-icon-preview.png`
  - `.codex/qa/settings-names-toggle-off.png`
  - `.codex/qa/settings-names-toggle-on.png`
  - `.codex/qa/rename-dialog-over-settings.png`
  - `.codex/qa/settings-name-button-after-dialog.png`
