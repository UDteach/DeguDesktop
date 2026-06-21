# ImageGen Source Prompt

Use this prompt shape for each coat:

```text
Create one production sprite sheet for a Windows desktop degu pet. Coat: <coat description> only. Strict fixed grid: 7 columns x 5 rows, transparent background, no text, no labels, no UI, no decorative background. All cells identical size with generous transparent padding; tail, whiskers, ears, and feet fully inside every cell. Row 1: idle breathing. Row 2: smooth walk cycle. Row 3: fast scurry cycle. Row 4: nibble/chew cycle. Row 5: startled hop cycle. Style: crisp cute pixel art, side view, 32-bit pixel style, consistent silhouette and scale across all 35 cells, long degu tail with tuft, big ears, whiskers, tiny paws.
```

For production assets, place one generated coat sheet per file under `assets/source/coats/`, then run:

```powershell
go run ./cmd/importsheet
```

There is no local art fallback. The ImageGen coat sheets are the source of truth.

Wheel-run frame prompt shape:

```text
Create one production-ready transparent PNG sprite frame for a Windows desktop pet app. Subject: one cute wild agouti degu only, side view facing right, running inside an exercise wheel posture, frame <n> of 6. No wheel, no cage, no prop, no text, no labels, no background, no shadow. Match the same camera, scale, pixel-art style, agouti coat, ear size, tail length, and silhouette across all six wheelrun frames. Fixed square transparent canvas with generous padding, full animal visible, crisp clean 32-bit pixel art, readable at 96x64 runtime size, no cropped ears/tail/feet/whiskers, no extra fragments.
```

Wheel source prompt:

```text
Create one production-ready transparent PNG asset for a Windows desktop pet app. Subject: a cute pixel-art degu exercise wheel back layer only, side view, no animal. The runtime draws the rotating front spokes separately, so this image must have a circular outer rim, subtle rear wooden running surface, small stable base/stand, open clear center, no front spokes, no hub cap, no animal, no text, no labels, no decorative background, no shadow. Crisp clean 32-bit pixel art, warm gray-brown wood and muted metal, readable at 72x72 pixels, centered on a square transparent canvas with generous padding. Full object visible, no cropped edges, no checkerboard baked into the image, no artifacts, no dirt, no extra parts.
```
