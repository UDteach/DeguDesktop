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

## Iteration 10 - Pages Build Metadata

Date: 2026-06-20

### Target

Show the downloadable build version, update date, and commit on the GitHub Pages download area.

### Cause

The download buttons did not expose which Pages-built binary a visitor would receive. That made it harder to tell whether the published page had caught up with the latest repository state.

### Pages Pass

- Added compact build metadata chips under the x64/x86/Releases download buttons.
- Added a disabled macOS placeholder chip to the download button row without adding a non-existent artifact link.
- Updated the Pages workflow to stamp `docs/index.html` at deploy time with `pages-<short sha>`, the JST deploy date, and the short commit ID.
- Reused the same short Pages version string for the x64 and x86 `main.appVersion` injection.

### Verification

- Stamping dry-run found exactly one `data-build-version`, `data-build-date`, and `data-build-commit` target in `docs/index.html`.
- `git diff --check`
- `go test -buildvcs=false ./...`
- `go vet -buildvcs=false ./...`

## Iteration 11 - macOS Bottom Overlay Port

Date: 2026-06-20

### Target

Port Degu Desktop to macOS in the same repository, with degus wandering along the bottom of the screen above the Dock.

### Cause

The repository previously had a Windows-only Win32 app plus a non-Windows stub that only printed that Degu Desktop was implemented for Windows. A real Mac release needs a darwin entry point, Cocoa host window, app-bundle packaging, release workflow artifacts, and clear platform-status docs.

### App Pass

- Added a darwin-specific Go entry point in `cmd/degu/main_darwin.go`.
- Added a small Objective-C Cocoa bridge in `cmd/degu/darwin_cocoa_darwin.m` and `cmd/degu/darwin_cocoa.h`.
- Created a transparent, click-through, always-on-top bottom overlay positioned above the Dock visible area.
- Added a menu-bar degu icon with Quit.
- Reused embedded generated degu sprite sheets so multiple coat variants wander along the bottom edge.
- Reused a transparent runtime sprite to generate the menu-bar icon, avoiding the non-transparent source icon background.
- Added keyboard reaction through macOS event monitoring when system permissions allow it.
- Kept Windows implementation isolated under the existing Windows build tag and changed the old non-Windows stub to exclude darwin.

### Packaging And Release

- Added `packaging/macos/Info.plist`.
- Added `scripts/build_macos.sh` to create an ad-hoc-signed `DeguDesktop.app` and zip it as `DeguDesktop-macos-<arch>.zip`.
- Kept the existing GitHub Release workflow unchanged because the current GitHub token lacks `workflow` scope; macOS arm64 and amd64 ZIPs are generated locally and can be uploaded to the Release with `gh release`.
- Updated `README.md`, `docs/index.html`, and `docs/development/current-state.md` with macOS status and release notes.

### Verification

- `plutil -lint packaging/macos/Info.plist`
- `bash -n scripts/build_macos.sh`
- `go test -buildvcs=false ./...`
- `go vet -buildvcs=false ./...`
- `go run ./cmd/importsheet`
- `GO_CMD=.codex/tools/go/bin/go GOARCH=arm64 VERSION=dev-local scripts/build_macos.sh`
- `GO_CMD=.codex/tools/go/bin/go GOARCH=amd64 VERSION=dev-local scripts/build_macos.sh`
- `GOOS=windows GOARCH=amd64 go build -buildvcs=false -ldflags="-H=windowsgui" -o dist/DeguDesktop.exe ./cmd/degu`
- `git diff --check`
- Launched the direct macOS binary and captured `.codex/qa/macos-degu-overlay.png`.
- Launched `dist/DeguDesktop.app` and captured `.codex/qa/macos-app-bundle-overlay.png`.
- Verified the menu-bar degu icon and captured `.codex/qa/macos-menu-bar-icon.png`.

### Remaining macOS Gaps

- No macOS settings window yet.
- No macOS click reaction, foraging behavior, update installer, or notarization automation yet.
- The local artifact is ad-hoc signed. Public distribution still needs Developer ID signing and Apple notarization before a polished external release.

## Iteration 12 - macOS Menu Settings And Motion Fix

Date: 2026-06-20

### Target

Fix the macOS animation that looked like it was looking around while moving, make the app icon show in the macOS app bundle, and let the menu-bar icon edit practical Mac settings.

### Cause

The macOS port was cycling every frame in the walk/scurry ranges, while Windows uses a stable subset to avoid generated-frame orientation jitter. The app bundle also lacked `CFBundleIconFile`, and the menu-bar item only exposed Quit.

### App Pass

- Matched macOS horizontal motion to the Windows stable walk frame sequence.
- Added Darwin tests for stable horizontal frames, direction movement, mirrored drawing, and menu-backed settings state.
- Added app-bundle `.icns` generation from the runtime degu sprite and wired `CFBundleIconFile`.
- Added menu-bar settings for speed, visible degu count, keyboard reaction, and exit.
- Persisted macOS menu settings under the user's Application Support config directory.

### Verification

- `go test -buildvcs=false ./cmd/degu`
- `go test -buildvcs=false ./...`
- `go vet -buildvcs=false ./...`
- `go run ./cmd/importsheet`
- `GO_CMD=.codex/tools/go/bin/go GOARCH=arm64 VERSION=v0.1.5 scripts/build_macos.sh`
- `GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go test -buildvcs=false -c -o /tmp/degu-windows-amd64.test.exe ./cmd/degu`
- `GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build -buildvcs=false -ldflags="-H=windowsgui -X main.appVersion=v0.1.5" -o /tmp/DeguDesktop.exe ./cmd/degu`
- `plutil -lint packaging/macos/Info.plist`
- `bash -n scripts/build_macos.sh`
- `codesign --verify --deep --strict /Applications/DeguDesktop.app`
- Opened the menu-bar icon through System Events and verified menu items: `速さ`, `表示数`, `キーボード反応`, `終了`.
- Verified menu actions save `speed`, `petCount`, and `wheelEnabled` to `~/Library/Application Support/DeguDesktop/settings.json`.
- Captured `.codex/qa/macos-direction-fix-bottom.png` and `.codex/qa/macos-menu-settings.png`.

### Remaining macOS Gaps

- No full macOS settings window yet.
- No macOS click reaction, foraging behavior, update installer, Developer ID signing, or notarization automation yet.

## Iteration 13 - macOS Names, Clicks, And Full Count Selection

Date: 2026-06-20

### Target

Bring the macOS build closer to Windows behavior by adding per-pet names, click reactions, and direct selection for 6-9 visible degus.

### Cause

The macOS settings window covered motion and coat controls, but did not yet expose the Windows name workflow. The macOS menu also only offered the older count shortcuts, so 6, 7, 8, and 9 were not directly selectable even though the runtime supports up to ten pets.

### App Pass

- Added macOS settings persistence for optional name labels and ten per-pet names.
- Added a native `名前` settings tab with a name-label toggle and one field per visible pet slot.
- Added cursor-hover name labels above degus when name labels are enabled.
- Added click reactions using a global left-click monitor while preserving the click-through desktop layer.
- Expanded the macOS menu-bar count menu and settings count popup to support every count from 1 through 10.
- Added Darwin tests for full visible-count support, name persistence, default display names, and click hit testing.

### Verification

- `clang -fsyntax-only -fblocks -x objective-c -framework Cocoa cmd/degu/darwin_cocoa_darwin.m`
- `go test -buildvcs=false ./cmd/degu`
- `go test -buildvcs=false ./...`
- `go vet -buildvcs=false ./...`
- `GO_CMD=.codex/tools/go/bin/go GOARCH=arm64 VERSION=v0.1.5 scripts/build_macos.sh`
- `GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go test -buildvcs=false -c -o /tmp/degu-windows-amd64.test.exe ./cmd/degu`
- `GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build -buildvcs=false -ldflags="-H=windowsgui -X main.appVersion=v0.1.5" -o /tmp/DeguDesktop.exe ./cmd/degu`
- `plutil -lint packaging/macos/Info.plist`
- `codesign --verify --deep --strict /Applications/DeguDesktop.app`
- Reinstalled `/Applications/DeguDesktop.app`, launched it, and verified the running process.
- Opened the menu-bar settings window through System Events and captured `.codex/qa/macos-settings-names-click-counts.png`.
- Opened the new `名前` tab and captured `.codex/qa/macos-settings-name-tab.png`.

### Remaining macOS Gaps

- Synthetic live-click QA was blocked by macOS accessibility permission for `osascript`; the click reaction hit test is covered in Go tests.
- No macOS foraging behavior, update installer, Developer ID signing, or notarization automation yet.

## Iteration 14 - macOS Big Sur Compatibility Build Path

Date: 2026-06-21

### Target

Create a macOS 11 Big Sur compatibility build path without requiring a real Big Sur test machine.

### Cause

Go 1.25 requires macOS 12 Monterey or later, so a Big Sur ZIP needs to be built with the last Go line that can run on macOS 11. The packaging script also previously hard-coded `LSMinimumSystemVersion` to `12.0`, which made every app bundle advertise Monterey or later even when a compatibility toolchain was used.

### Packaging Pass

- Lowered the module `go` directive to `1.24.0`.
- Downgraded `golang.org/x/image` to `v0.36.0`, the latest checked version in this pass with a Go 1.24 module directive.
- Added `MACOS_MIN_VERSION` to `scripts/build_macos.sh` and template `LSMinimumSystemVersion` into `Info.plist`.
- Added `MACOS_COMPAT_LABEL` so Big Sur ZIPs are named `DeguDesktop-macos-big-sur-<arch>.zip` without changing the default Monterey-or-later ZIP names.
- Exported `MACOSX_DEPLOYMENT_TARGET` and cgo `-mmacosx-version-min` flags so the Mach-O `LC_BUILD_VERSION` records the requested minimum OS.
- Built the generated `.icns` helper for the host architecture, which keeps amd64 cross-packaging working on Apple Silicon hosts.
- Documented the Big Sur compatibility commands and release asset names in `README.md` and the current-state file.

### Verification

- `GOTOOLCHAIN=local .codex/tools/go1.24.11/bin/go test -buildvcs=false ./...`
- `GOTOOLCHAIN=local .codex/tools/go1.24.11/bin/go vet -buildvcs=false ./...`
- `.codex/tools/go/bin/go test -buildvcs=false ./...`
- `.codex/tools/go/bin/go vet -buildvcs=false ./...`
- `GOTOOLCHAIN=local GO_CMD=.codex/tools/go1.24.11/bin/go GOARCH=amd64 VERSION=v0.1.5-big-sur MACOS_MIN_VERSION=11.0 MACOS_COMPAT_LABEL=big-sur scripts/build_macos.sh`
- `GOTOOLCHAIN=local GO_CMD=.codex/tools/go1.24.11/bin/go GOARCH=arm64 VERSION=v0.1.5-big-sur MACOS_MIN_VERSION=11.0 MACOS_COMPAT_LABEL=big-sur scripts/build_macos.sh`
- Verified both Big Sur ZIPs have `LSMinimumSystemVersion=11.0`, matching `CFBundleShortVersionString=v0.1.5-big-sur`, the expected Mach-O architecture, and `LC_BUILD_VERSION minos 11.0`.
- `codesign --verify --deep --strict` passed for both extracted Big Sur app bundles.
- `GO_CMD=.codex/tools/go/bin/go GOARCH=arm64 VERSION=v0.1.5 scripts/build_macos.sh`
- Verified the default macOS ZIP still has `LSMinimumSystemVersion=12.0` and `LC_BUILD_VERSION minos 12.0`.
- `GOOS=windows GOARCH=amd64 CGO_ENABLED=0 .codex/tools/go/bin/go test -buildvcs=false -c -o /tmp/degu-windows-amd64.test.exe ./cmd/degu`
- `GOOS=windows GOARCH=amd64 CGO_ENABLED=0 .codex/tools/go/bin/go build -buildvcs=false -ldflags="-H=windowsgui -X main.appVersion=v0.1.5" -o /tmp/DeguDesktop.exe ./cmd/degu`

### Remaining Risk

- Big Sur support is statically checked and packaged, but not smoke-tested on a real macOS 11 machine.
- Public distribution still needs Developer ID signing and notarization as release-operator steps.

## Iteration 15 - Wheel Rim Fit And Dedicated Wheel-Run Frames

Date: 2026-06-21

### Target

Make the typing wheel look physically coherent by keeping the degu inside the wheel rim and replacing the normal walk-cycle reuse with dedicated wheel-running frames.

### Cause

The previous wheel source was a complete wheel illustration while the runtime also drew rotating front spokes and a hub. The degu inside the wheel also reused `walkFrameSeq`, so feet and tail could look like they belonged to ground walking rather than wheel running, and the original 68x46 runner draw size visibly protruded outside the rim.

### Asset Pass

- Generated a new ImageGen wheel back-layer source with an open center, rim, rear running surface, and stable base.
- Added `cleanWheelArtwork` so enclosed baked checker pixels inside the wheel opening become transparent.
- Generated six separate ImageGen source PNGs for `wheelrun`.
- Added them as `assets/source/frames/wild_agouti/56_wheelrun_00.png` through `61_wheelrun_05.png`.
- Expanded the runtime frame contract from 56 to 62 frames.
- Updated Windows `stateWheel` and macOS keyboard-wheel rendering to use `wheelRunFrameSeq` instead of `walkFrameSeq`.
- Tuned wheel runner drawing from 68x46+6px to 56x38+2px so all coat variants visually stay inside the rim.
- Updated source docs and prompts to define the wheel as a back layer and `wheelrun` as one-frame-per-image production input.

### Verification

- `go run ./cmd/importsheet`
- QA images:
  - `.codex/qa/wheel-72px-zoom.png`
  - `.codex/qa/wheelrun-source-contact.png`
  - `.codex/qa/wheelrun-normalized-contact.png`
  - `.codex/qa/wheelrun-fit-candidates.png`
  - `.codex/qa/wheelrun-all-colors-composite.png`
- Final wheel-run fit audit across all 11 coat variants had a max outside-inner-wheel ratio of `0.0000`, visually contained within the wheel rim.
- `go test -buildvcs=false ./cmd/importsheet`
- `go test -buildvcs=false ./cmd/degu`
- `go test -buildvcs=false ./...`
- `go vet -buildvcs=false ./...`
- Local Windows x64/x86 build with `main.appVersion=v0.1.6`
- `git diff --check`

## Iteration 16 - macOS v0.1.6 Release Sync

Date: 2026-06-21

### Target

Bring the macOS release artifacts up to the Windows `v0.1.6` release.

### Release Pass

- Built `DeguDesktop-macos-arm64.zip` and `DeguDesktop-macos-amd64.zip` with `VERSION=v0.1.6`.
- Built Big Sur compatibility ZIPs `DeguDesktop-macos-big-sur-arm64.zip` and `DeguDesktop-macos-big-sur-amd64.zip` with Go 1.24.11, `VERSION=v0.1.6-big-sur`, and `MACOS_MIN_VERSION=11.0`.
- Uploaded all four macOS ZIPs to the existing GitHub Release `v0.1.6`.
- Updated the GitHub Release notes with macOS download guidance.
- Updated the GitHub Pages Mac direct links and visible Mac version from `v0.1.5` to `v0.1.6`.

### Verification

- Verified macOS 12+ ZIPs have `CFBundleShortVersionString=v0.1.6`, `LSMinimumSystemVersion=12.0`, and Mach-O `LC_BUILD_VERSION minos 12.0`.
- Verified Big Sur ZIPs have `CFBundleShortVersionString=v0.1.6-big-sur`, `LSMinimumSystemVersion=11.0`, and Mach-O `LC_BUILD_VERSION minos 11.0`.
- `codesign --verify --deep --strict` passed for all four extracted macOS app bundles.
- `.codex/tools/go/bin/go test -buildvcs=false ./...`
- `.codex/tools/go/bin/go vet -buildvcs=false ./...`
- `GOTOOLCHAIN=local .codex/tools/go1.24.11/bin/go test -buildvcs=false ./...`
- `GOTOOLCHAIN=local .codex/tools/go1.24.11/bin/go vet -buildvcs=false ./...`
- Local Windows amd64 GUI cross-build with `main.appVersion=v0.1.6`.

## Iteration 17 - Windows Overlay Position Controls

Date: 2026-06-21

### Target

Let users adjust the default Windows overlay position so degus do not have to float above the taskbar, and support setups where the taskbar is attached to the left edge.

### Cause

The Windows overlay used the work-area bottom directly for drawing, hit testing, and hover-name placement. That kept the app above a bottom taskbar and also made left-edge taskbar layouts inherit the reduced work-area width. Users need a saved way to choose a physical screen-bottom baseline and fine-tune the vertical offset.

### Implementation

- Added saved Windows settings for overlay position mode and vertical offset.
- Added motion-tab controls for `Taskbar edge`, `Screen bottom`, `Up`, and `Down`.
- Added a default +10 px downward offset for legacy settings that do not yet contain position fields.
- Unified render placement, click hit testing, and hover-name positioning through the same overlay rectangle calculation.
- Added primary-screen bottom placement so left-taskbar users can choose full screen-width bottom alignment.
- Made the settings reset button restore the overlay position defaults.

### Verification

- Added tests for settings round-trip persistence, legacy settings defaults, and taskbar-vs-screen-bottom overlay rectangles.
- `go test -buildvcs=false ./cmd/degu`
- `go test -buildvcs=false ./...`
- `go vet -buildvcs=false ./...`
- `go build -buildvcs=false -ldflags="-H=windowsgui" -o dist\DeguDesktop.exe ./cmd/degu`

## Iteration 18 - Tray Count, Pet Alignment Help, And v0.1.7 QA

Date: 2026-06-21

### Target

Make the Windows tray count menu cover 6, 7, 8, and 9 as requested, explain why some degus appear slightly higher, let users choose same-baseline alignment, and prepare a v0.1.7 release with Pages updated.

### Cause

The tray menu used a fixed shortcut list of 1, 2, 3, 5, and 10. The runtime also staggered pet `laneOffset` values by 0/5/10 px so overlapping degus were easier to distinguish, but that behavior was not visible as a setting and looked like accidental vertical drift.

### Implementation

- Replaced fixed tray count commands with generated 1-10 count commands.
- Added saved pet-height alignment with `Natural stagger` and `Same baseline` choices.
- Added settings hover explanations in the page lead area, with standard Win32 tooltips still registered for controls.
- Renamed the ambiguous Japanese reset label from `整列` to `配置リセット`.
- Added update-check tests for GitHub release JSON validation, HTTP errors, draft releases, download, and update ZIP extraction.
- Updated the GitHub Pages copy and Windows release label to `v0.1.7`.

### Verification

- `go test -buildvcs=false ./cmd/degu`
- `go test -buildvcs=false ./...`
- `go vet -buildvcs=false ./...`
- `go run ./cmd/importsheet`
- `git diff --check`
- Built local Windows x64/x86 ZIPs with `main.appVersion=v0.1.7` and verified both contain `DeguDesktop.exe` and `README.md`.
- Recreated release ZIPs from scratch and verified PE machine types: amd64 `0x8664`, x86 `0x014c`.
- Visual settings QA screenshots:
  - `.codex/qa/settings-animals-v017-hoverfix.png`
  - `.codex/qa/settings-motion-clear-v017.png`
  - `.codex/qa/settings-motion-hover-v017.png`

## Iteration 19 - macOS v0.1.7 Release Sync

Date: 2026-06-21

### Target

Bring the macOS release artifacts up to the Windows `v0.1.7` release.

### Release Pass

- Built `DeguDesktop-macos-arm64.zip` and `DeguDesktop-macos-amd64.zip` with `VERSION=v0.1.7`.
- Built Big Sur compatibility ZIPs `DeguDesktop-macos-big-sur-arm64.zip` and `DeguDesktop-macos-big-sur-amd64.zip` with Go 1.24.11, `VERSION=v0.1.7-big-sur`, and `MACOS_MIN_VERSION=11.0`.
- Uploaded all four macOS ZIPs to the existing GitHub Release `v0.1.7`.
- Replaced the release note that pointed Mac users to `v0.1.6` with macOS `v0.1.7` download guidance.
- Updated the GitHub Pages Mac direct links and visible Mac version from `v0.1.6` to `v0.1.7`.

### Verification

- Verified macOS 12+ ZIPs have `CFBundleShortVersionString=v0.1.7`, `LSMinimumSystemVersion=12.0`, and Mach-O `LC_BUILD_VERSION minos 12.0`.
- Verified Big Sur ZIPs have `CFBundleShortVersionString=v0.1.7-big-sur`, `LSMinimumSystemVersion=11.0`, and Mach-O `LC_BUILD_VERSION minos 11.0`.
- `codesign --verify --deep --strict` passed for all four extracted macOS app bundles.
- `.codex/tools/go/bin/go test -buildvcs=false ./...`
- `.codex/tools/go/bin/go vet -buildvcs=false ./...`
- `GOTOOLCHAIN=local .codex/tools/go1.24.11/bin/go test -buildvcs=false ./...`
- `GOTOOLCHAIN=local .codex/tools/go1.24.11/bin/go vet -buildvcs=false ./...`
- Local Windows amd64 GUI cross-build with `main.appVersion=v0.1.7`.

## Iteration 20 - Random Stroll Wheel Choice

Date: 2026-06-21

### Target

Let random stroll mode occasionally use the wheel without requiring keyboard input.

### Cause

The wheel state was only entered from `onTyping()`, and `onTyping()` intentionally ignores input while the app is in random stroll mode. That meant the wheel never appeared during random stroll even when the wheel option was enabled.

### Implementation

- Added a low-probability random wheel action to `chooseRandomAction`.
- Kept only one wheel runner active at a time.
- Gave random wheel sessions a longer hold time than typing-triggered wheel sessions.
- Preserved the existing behavior that typing itself does not trigger the wheel in random stroll mode.

### Verification

- Added tests for random wheel entry, mode/setting/active-runner guards, and the existing typing guard in random mode.
- `gofmt -w cmd\degu\main_windows.go cmd\degu\motion_windows_test.go`
- `go test -buildvcs=false ./...`
- `go vet -buildvcs=false ./...`
- `git diff --check`
- `go build -buildvcs=false -ldflags="-H=windowsgui -s -w -X main.appVersion=v0.1.7-random-wheel-local" -o dist\DeguDesktop.exe ./cmd/degu`

## Iteration 21 - v0.1.8 Release And Pages Refresh

Date: 2026-06-21

### Target

Publish the random stroll wheel behavior as `v0.1.8` and refresh the GitHub Pages download copy.

### Cause

Iteration 20 only rebuilt and launched a local test binary. The public Windows download and Pages version label still pointed at `v0.1.7`. The Mac links should remain on the existing `v0.1.7` Mac assets from Iteration 19.

### Implementation

- Updated README and GitHub Pages copy to describe wheel use during typing and occasional random stroll.
- Updated the Pages Windows latest label to `v0.1.8`.
- Initially kept Pages Mac download links and label on the existing `v0.1.7` Mac assets until matching macOS artifacts were built.

### Verification

- `go run ./cmd/importsheet`
- `go test -buildvcs=false ./...`
- `go vet -buildvcs=false ./...`
- `git diff --check`
- Built local Windows x64/x86 ZIPs with `main.appVersion=v0.1.8`.
- Verified ZIP contents include `DeguDesktop.exe` and `README.md`.
- Verified PE machine types: amd64 `0x8664`, x86 `0x014c`.
- Pushed `main` and tag `v0.1.8`.
- GitHub Release workflow `27898483118` completed successfully and published `DeguDesktop-windows-amd64.zip` and `DeguDesktop-windows-386.zip`.
- GitHub Pages workflow `27898530576` completed successfully.
- Verified live GitHub Pages showed Windows `v0.1.8`, Mac `v0.1.7`, build commit `2365b87`, and the random stroll wheel copy before the later macOS sync.
- Verified GitHub Releases API reports `v0.1.8` as latest with both Windows assets.

## Iteration 22 - macOS v0.1.8 Release Sync

Date: 2026-06-21

### Target

Bring the macOS release artifacts up to the Windows `v0.1.8` release.

### Release Pass

- Built `DeguDesktop-macos-arm64.zip` and `DeguDesktop-macos-amd64.zip` with `VERSION=v0.1.8`.
- Built Big Sur compatibility ZIPs `DeguDesktop-macos-big-sur-arm64.zip` and `DeguDesktop-macos-big-sur-amd64.zip` with Go 1.24.11, `VERSION=v0.1.8-big-sur`, and `MACOS_MIN_VERSION=11.0`.
- Uploaded all four macOS ZIPs to the existing GitHub Release `v0.1.8`.
- Updated the GitHub Release notes with macOS `v0.1.8` download guidance.
- Updated the GitHub Pages Mac direct links and visible Mac version from `v0.1.7` to `v0.1.8`.

### Verification

- Verified macOS 12+ ZIPs have `CFBundleShortVersionString=v0.1.8`, `LSMinimumSystemVersion=12.0`, and Mach-O `LC_BUILD_VERSION minos 12.0`.
- Verified Big Sur ZIPs have `CFBundleShortVersionString=v0.1.8-big-sur`, `LSMinimumSystemVersion=11.0`, and Mach-O `LC_BUILD_VERSION minos 11.0`.
- `codesign --verify --deep --strict` passed for all four extracted macOS app bundles.
- `.codex/tools/go/bin/go test -buildvcs=false ./...`
- `.codex/tools/go/bin/go vet -buildvcs=false ./...`
- `GOTOOLCHAIN=local .codex/tools/go1.24.11/bin/go test -buildvcs=false ./...`
- `GOTOOLCHAIN=local .codex/tools/go1.24.11/bin/go vet -buildvcs=false ./...`
- Local Windows amd64 GUI cross-build with `main.appVersion=v0.1.8`.

## Iteration 23 - Display Tab And Taskbar Walking Range

Date: 2026-06-21

### Target

Let users choose which display the Windows pet overlay uses and set a clear "from here to here" walking range over the taskbar area, with visible scrollbars in the settings window.

### Cause

The existing display controls were mixed into the motion tab and only adjusted vertical placement. Users with a left taskbar or multiple displays needed a more explicit display section, and there was no way to constrain horizontal roaming to a specific taskbar segment.

### Implementation

- Added a dedicated Windows Display tab to the Win32 settings panel.
- Added saved `displayIndex`, `walkRangeStart`, and `walkRangeEnd` settings.
- Added monitor enumeration with primary-first ordering and virtual-screen fallback.
- Added two visible horizontal scrollbar controls for the walking range start/end, plus full/narrow/wide/left/right buttons.
- Applied the walking range to the overlay bounds and clamped pets/forage into the updated scene width.
- Moved display position and pet-height alignment controls from Motion to Display.
- Fixed a QA-found crash where monitor enumeration created a new Windows callback every frame and eventually hit Go's callback limit.

### Verification

- `gofmt -w cmd\degu\main_windows.go cmd\degu\motion_windows_test.go`
- `go test -buildvcs=false ./cmd/degu`
- `go test -buildvcs=false ./...`
- `go vet -buildvcs=false ./...`
- `go run ./cmd/importsheet`
- `go build -buildvcs=false -ldflags="-H=windowsgui" -o dist\DeguDesktop.exe ./cmd/degu`
- `git diff --check`
- Added tests for settings persistence, walk-range bounds, negative monitor coordinates, repeated monitor enumeration, and Display-tab tooltips.
- Launched `dist\DeguDesktop.exe`, opened Settings through the normal command handler, switched to Display, verified the app stayed running, and captured:
  - `.codex/qa/settings-display-range-final.png`
  - `.codex/qa/settings-display-range-narrow.png`

## Iteration 24 - Updates Settings Tab

Date: 2026-06-21

### Target

Make the existing GitHub Release update support visible and understandable from the settings window, not only the tray menu.

### Cause

The updater already supported release checking, matching Windows x64/x86 zip selection, download, installer staging, and restart. However, the primary settings surface did not show update state, current/latest version, matching package, or install action, so users had to discover it from the tray menu.

### Implementation

- Added a dedicated Updates tab to the Win32 settings panel.
- Added update status, package, and action sections with Japanese/English text.
- Wired the settings buttons to the existing `startUpdateCheck(true)` and `installLatestUpdate()` paths.
- Disabled the check/install buttons while update work is running, and disabled install until a matching newer release asset is available.
- Synced the settings window when async update checks complete or fail.
- Added tests for update summary states, package size display, installability, and update-tab tooltips.

### Verification

- `gofmt -w cmd\degu\main_windows.go cmd\degu\motion_windows_test.go`
- `go test -buildvcs=false ./cmd/degu`
- `go test -buildvcs=false ./...`
- `go vet -buildvcs=false ./...`
- `go build -buildvcs=false -ldflags="-H=windowsgui" -o dist\DeguDesktop.exe ./cmd/degu`
- `git diff --check`
- Launched `dist\DeguDesktop.exe`, opened Settings through the normal command handler, switched to Updates, verified the app stayed running, and captured:
  - `.codex/qa/settings-updates-tab.png`
  - `.codex/qa/settings-updates-tab-result.png`

## Iteration 25 - Settings Home Overview

Date: 2026-06-21

### Target

Make the settings window easier to understand on first open by adding a MofuMouse-style overview entry point.

### Cause

After adding Display and Updates, the settings surface had the needed controls but still opened directly into a single detail section. Users needed a clearer landing view that summarizes current state and points them to the right category without guessing.

### Implementation

- Added a Home tab as the default settings tab.
- Added four overview cards: Degu, Motion, Display, and Updates.
- Added one-click Open buttons from Home into each detailed settings section.
- Added concise summaries for coat/name state, motion mode/speed/wheel/turns, display/range/position, and update/package state.
- Added Japanese/English labels and tooltips for the Home tab and shortcuts.

### Verification

- `gofmt -w cmd\degu\main_windows.go cmd\degu\motion_windows_test.go`
- `go test -buildvcs=false ./cmd/degu`
- Built `dist\DeguDesktop.exe`, opened Settings through the normal command handler, verified Home renders first, then used the Display shortcut and confirmed the app stayed running.
- Visual QA screenshot:
  - `.codex/qa/settings-home-tab.png`

## Iteration 26 - v0.1.9 Version Bump Prep

Date: 2026-06-21

### Target

Prepare the Windows release line for `v0.1.9` after the display range, update tab, and home settings improvements.

### Cause

The current public Windows release label was still `v0.1.8`, while the local app now includes the new user-facing settings and display behavior.

### Implementation

- Updated the GitHub Pages Windows latest label to `v0.1.9`.
- Rebuilt local Windows x64/x86 download ZIPs with `main.appVersion=v0.1.9`.
- Initially kept macOS links on the existing `v0.1.8` assets until matching macOS release artifacts were built in the follow-up sync.

### Verification

- `go test -buildvcs=false ./...`
- `go vet -buildvcs=false ./...`
- `go run ./cmd/importsheet`
- `git diff --check`
- Verified both local ZIPs contain exactly `DeguDesktop.exe` and `README.md`.
- Verified x64 ZIP contents use PE machine `0x8664`, x86 ZIP contents use PE machine `0x014c`, and both staged executables contain `v0.1.9`.
- Replaced and relaunched the local `dist\DeguDesktop.exe` from the x64 `v0.1.9` build.

### Release Status

- Pushed `main` and tag `v0.1.9`.
- GitHub Release workflow `27918389775` completed successfully and published `DeguDesktop-windows-amd64.zip` and `DeguDesktop-windows-386.zip`.
- GitHub Pages workflow `27918386994` completed successfully.
- GitHub Pages workflow `27918482902` completed successfully after the release-verification log commit.
- Verified live GitHub Pages initially showed Windows `v0.1.9`, Mac `v0.1.8`, and Pages build metadata stamped from the latest deployed commit before the later macOS sync.
- Verified GitHub Releases reports `v0.1.9` as latest with both Windows assets.
- Verified latest download URLs for both Windows x64 and x86 return HTTP 200.

## Iteration 27 - macOS v0.1.9 Release Sync

Date: 2026-06-22

### Target

Bring the macOS release artifacts and download links up to the Windows `v0.1.9` release.

### Cause

The `v0.1.9` GitHub Release had the Windows x64/x86 ZIPs, while the public Mac links still pointed at the previous `v0.1.8` assets.

### Implementation

- Built `DeguDesktop-macos-arm64.zip` and `DeguDesktop-macos-amd64.zip` with `VERSION=v0.1.9`.
- Built Big Sur compatibility ZIPs `DeguDesktop-macos-big-sur-arm64.zip` and `DeguDesktop-macos-big-sur-amd64.zip` with Go 1.24.11, `VERSION=v0.1.9-big-sur`, and `MACOS_MIN_VERSION=11.0`.
- Uploaded all four macOS ZIPs to the existing GitHub Release `v0.1.9`.
- Updated the GitHub Release notes with macOS `v0.1.9` download guidance.
- Updated the GitHub Pages Mac direct links and visible Mac version from `v0.1.8` to `v0.1.9`.
- Updated the README Big Sur compatibility build examples to `v0.1.9-big-sur`.

### Verification

- Verified macOS 12+ ZIPs have `CFBundleShortVersionString=v0.1.9`, `CFBundleVersion=v0.1.9`, `LSMinimumSystemVersion=12.0`, and Mach-O `LC_BUILD_VERSION minos 12.0`.
- Verified Big Sur ZIPs have `CFBundleShortVersionString=v0.1.9-big-sur`, `CFBundleVersion=v0.1.9-big-sur`, `LSMinimumSystemVersion=11.0`, and Mach-O `LC_BUILD_VERSION minos 11.0`.
- Verified all four ZIPs are ad-hoc signed and do not contain AppleDouble metadata.
- `go test -buildvcs=false ./...`
- `go vet -buildvcs=false ./...`
- Go 1.24.11 compatibility `go test -buildvcs=false ./...`
- Go 1.24.11 compatibility `go vet -buildvcs=false ./...`
- Local Windows amd64 GUI cross-build with `main.appVersion=v0.1.9`.

### Remaining Risk

- Big Sur support is statically checked and packaged, but not smoke-tested on a real macOS 11 machine.

## Iteration 28 - Pages Settings Guide

Date: 2026-06-22

### Target

Explain how to open and use settings from the Windows task tray on the GitHub Pages site, with settings screenshots.

### Cause

The download page described the app features but did not clearly show new users where the settings window is opened or what the main settings tabs look like.

### Implementation

- Added a Settings link to the page navigation.
- Added a Japanese-first settings guide section with task-tray steps: launch, right-click the tray icon, open Settings, then adjust display options.
- Added settings screenshots for the Home, Display/walking range, and Updates tabs.
- Added a short English settings note.

### Verification

- Copied and visually checked the three settings screenshots.
- Captured local desktop and mobile page screenshots:
  - `.codex/qa/pages-settings-section-desktop.png`
  - `.codex/qa/pages-settings-section-mobile.png`
- Verified local HTML image references and section anchors.
- `git diff --check`

## Iteration 29 - Tray Temporary Hide

Date: 2026-06-25

### Target

Add a task-tray right-click action that can temporarily hide the visible degus without quitting the app.

### Cause

Users need a quick way to clear the desktop pet overlay for the current session while keeping the tray icon, settings, and later restore path available.

### Implementation

- Added a non-persisted runtime visibility flag to the Windows app.
- Added a tray menu item that switches between Japanese/English "temporarily hide" and "show pets" labels.
- Hid the overlay and name label window while hidden, and restored the overlay when the same menu item is selected again.
- Blocked keyboard and click pet reactions while the overlay is temporarily hidden.
- Added tests for menu labels and the hidden-state typing guard.

### Verification

- `gofmt -w cmd\degu\main_windows.go cmd\degu\motion_windows_test.go cmd\importsheet\main.go cmd\importsheet\main_test.go`
- `go test -buildvcs=false ./...`
- `go vet -buildvcs=false ./...`
- `go run ./cmd/importsheet`
- `go build -buildvcs=false -ldflags="-H=windowsgui" -o dist\DeguDesktop.exe ./cmd/degu`
- `git diff --check`

## Iteration 30 - Pages Version History

Date: 2026-06-25

### Target

Add a concise version history to the GitHub Pages download page so users can see what changed in recent releases.

### Cause

The download page showed the current version and links, but it did not explain what each recent version fixed or added. Users had to open GitHub Releases or development logs to understand release differences.

### Implementation

- Added a `バージョン履歴` navigation link and section near the download area.
- Listed public releases `v0.1.9` through `v0.1.5` with short Japanese summaries.
- Added a GitHub Releases link for older or detailed release information.
- Kept unreleased local changes out of the public history to avoid implying they are already in the downloadable release.
- Added responsive CSS for desktop two-column history rows and single-column mobile rows.

### Verification

- Rendered `docs/index.html` with Playwright using local Chrome at desktop and mobile widths.
- Confirmed the history section exists, the navigation contains `バージョン履歴`, the version list is `v0.1.9` through `v0.1.5`, and there is no horizontal overflow.
- Captured screenshots:
  - `.codex/qa/pages-version-history-desktop.png`
  - `.codex/qa/pages-version-history-mobile.png`
- `git diff --check`

## Iteration 31 - Multi-Monitor Display Spans

Date: 2026-06-25

### Target

Let Windows users place Degu Desktop across a selected span of multiple monitors, not only one monitor at a time.

### Cause

The Display tab previously selected a single monitor. Users with two or three monitors need the pet overlay to move across more than one screen, and the existing walking-range percentage controls should apply to that selected multi-monitor span.

### Implementation

- Added display scope state for single-display mode and multi-display span mode.
- Added persisted `displayScope` and `displaySpanEnd` settings while keeping legacy single-monitor settings compatible.
- Added Display-tab controls for `1画面`, `複数画面`, span shrinking/expanding, and moving the selected span left/right.
- Combined selected monitor rectangles into one overlay range, with taskbar-edge placement using the shared work-area bottom where possible.
- Applied the existing walking-range percentage controls to the selected single or multi-monitor span.
- Added monitor-boundary markers to the walking-range preview when a multi-monitor span is active.
- Updated README, GitHub Pages copy, and current-state docs.

### Verification

- `go test -buildvcs=false ./cmd/degu`
- `go test -buildvcs=false ./...`
- `go vet -buildvcs=false ./...`
- `go run ./cmd/importsheet`
- `go build -buildvcs=false -ldflags="-H=windowsgui" -o dist\DeguDesktop.exe ./cmd/degu`
- `git diff --check`
- Added tests for 3-monitor span selection and walking-range application across selected monitor spans.
- Rendered `docs/index.html` with Playwright using local Chrome at desktop and mobile widths, confirming no horizontal overflow after the copy update.
- Launched `dist\DeguDesktop.exe`, opened Settings through the normal command handler, switched to Display, verified the app exited cleanly, and captured:
  - `.codex/qa/settings-display-multimonitor-span.png`
  - `.codex/qa/pages-multimonitor-copy-desktop.png`
  - `.codex/qa/pages-multimonitor-copy-mobile.png`

## Iteration 32 - v0.1.10 Release Prep

Date: 2026-06-25

### Target

Prepare and publish the Windows `v0.1.10` release after the tray temporary-hide action, Pages version history, and multi-monitor span support.

### Cause

The latest public Windows release is `v0.1.9`, while `main` now contains user-facing fixes for temporary hiding and multi-monitor placement that should be available through GitHub Releases and the Pages download page.

### Implementation

- Updated the GitHub Pages Windows latest label to `v0.1.10`.
- Added a `v0.1.10` entry to the public version history.
- Documented that Mac download links remain on the existing `v0.1.9` artifacts until a separate macOS sync is built.
- Prepared to tag `v0.1.10` so the existing Release workflow can publish Windows x64 and x86 ZIPs.

### Verification

- `gofmt -w cmd\degu\main_windows.go cmd\degu\motion_windows_test.go cmd\importsheet\main.go cmd\importsheet\main_test.go`
- `go test -buildvcs=false ./...`
- `go vet -buildvcs=false ./...`
- `go run ./cmd/importsheet`
- Built local Windows amd64 and 386 executables with `main.appVersion=v0.1.10`.
- Verified local PE machine values: amd64 `0x8664`, 386 `0x014c`.
- Verified local executables contain the embedded `v0.1.10` string.
- Rendered the local GitHub Pages history section with Playwright at desktop and mobile widths, and captured:
  - `.codex/qa/pages-v0.1.10-history-desktop.png`
  - `.codex/qa/pages-v0.1.10-history-mobile.png`
- `git diff --check`
- Pushed `main` and tag `v0.1.10`.
- Verified GitHub Actions Release run `28161539683` completed successfully.
- Verified GitHub Actions Pages run `28161538316` completed successfully.
- Verified GitHub Release `v0.1.10` is marked Latest and contains:
  - `DeguDesktop-windows-amd64.zip`
  - `DeguDesktop-windows-386.zip`
- Downloaded both published ZIPs from GitHub Releases and verified their contents:
  - amd64 EXE uses PE machine `0x8664` and contains `v0.1.10`.
  - 386 EXE uses PE machine `0x014c` and contains `v0.1.10`.
  - both ZIPs include `DeguDesktop.exe` and `README.md`.
- Verified the live GitHub Pages site shows Windows latest `v0.1.10`, includes the version history section, and is stamped with release-prep commit `c6d2346`.
- Verified `releases/latest/download` redirects to `v0.1.10` for both Windows x64 and x86 ZIPs.
