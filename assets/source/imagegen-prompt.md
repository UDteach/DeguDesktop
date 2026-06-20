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

Wheel source prompt:

```text
Create a single transparent PNG asset for a Windows desktop pet app: a cute pixel-art degu exercise wheel, side view, no animal, no background, no text, no labels. Crisp 32-bit pixel art style, warm gray-brown wooden/metal wheel, circular rim, simple spokes, small stable base, centered with generous transparent padding. Fixed square canvas, full object visible, no cropped edges, no extra artifacts, no decorative background.
```
