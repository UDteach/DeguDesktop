# ImageGen Asset Intake

Preferred source is one ImageGen PNG per runtime frame:

- path: `assets/source/frames/<coat_id>/<frame>_<action>_<step>.png`
- 62 files for the canonical `wild_agouti` motion set
- actions: idle, walk, scurry, nibble, hop, turn, eat, dig, stand, groomface, wheelrun
- transparent background or simple checker background
- one complete degu per file, with ears, whiskers, feet, and tail fully inside the image

Fallback ImageGen sheets are still supported:

- `assets/source/imagegen-idle.png`
- `assets/source/imagegen-walk.png`
- `assets/source/imagegen-scurry.png`
- `assets/source/imagegen-nibble.png`
- `assets/source/imagegen-hop.png`

Extra ImageGen source assets:

- `assets/source/imagegen-wheel.png` - transparent pixel-art exercise wheel back-layer source; front spokes are drawn at runtime
- `assets/source/imagegen-icon.png` - square pixel-art degu icon source

Importer:

```powershell
go run ./cmd/importsheet
```

The importer writes:

- `assets/sprites/degu_*.png`
- `assets/tray.ico`
- `docs/assets/degu-preview.png`
- `assets/source/import-report.json`

The report warns when background removal finds no content or when source content touches an edge and may be cropped.
