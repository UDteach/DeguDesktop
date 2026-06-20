# Art Direction

## Target

Build readable pixel-style taskbar pets for three distinct species:

- Degu
- Chinchilla
- Macaroni mouse, also cross-checked as fat-tailed gerbil to reduce species confusion

Species must be recognizable by silhouette, not just color.

## Shared Frame Rules

- Transparent background.
- One complete animal only.
- Strict side view.
- Consistent camera and scale.
- Consistent anatomy across frames.
- Complete ears, feet, whiskers, and tail.
- No text.
- No border.
- No scenery.
- No cast shadow.
- No multiple animals.
- No cropped body parts.
- No costume.
- No human-like pose.
- Readable pixel-art silhouette at taskbar size.

## Degu Notes

- Medium-sized body.
- Slim, slightly long body.
- Large rounded ears.
- Long thin tail with a tuft at the tip.
- Quick movement, but not as tiny or fast as a mouse.
- Walking, stopping, nibbling, hopping, and running should have distinct poses.
- Turning should use transitional three-quarter poses, not sudden left/right sprite flips.
- Blue coat variants should read as low-saturation slate/greige, not vivid blue.
- Pied variants need irregular ImageGen-derived white patch placement, not simple recolors or oval masks.

## Chinchilla Notes

- Larger than degu.
- Rounder, higher-density body.
- Very large rounded ears.
- Shorter muzzle.
- Thick fluffy tail.
- Heavier body rhythm while walking.
- Hop is higher than degu, with a softer landing.

## Macaroni Mouse Notes

- Smallest of the three.
- Short rounded body.
- Large dark eyes.
- Small feet and hands.
- Long distinctive thick tail, not a normal thin mouse tail.
- Fine, quick walking.
- Low rounded idle posture.
- Seed-carrying behavior should suit the species.

## Generation Prompt Template

Use this as a base for future ImageGen work:

```text
transparent background, one complete [species] only, strict side view,
consistent camera and scale, consistent anatomy across frames,
complete ears, complete feet, complete whiskers, complete tail,
pixel-art sprite, readable silhouette at Windows taskbar size,
no text, no border, no scenery, no cast shadow, no multiple animals,
no cropped body parts, no costume, no human-like pose
```

## Turn Prompt Template

Use one prompt per frame for turn animation work:

```text
transparent background, one complete wild agouti degu only,
TURN ANIMATION FRAME [n] OF 8, right-to-left turn,
subtle three-quarter transition, not front-facing, not staring at viewer,
consistent body size, consistent low foot baseline,
complete ears, complete feet, complete whiskers, complete long thin tail with tuft,
pixel-art sprite, readable silhouette at Windows taskbar size,
no text, no border, no scenery, no cast shadow, no multiple animals,
no cropped body parts
```

## Pied Coat Guide Prompt Template

Use one source image per pied coat guide:

```text
transparent background, one complete [coat color] pied degu only,
strict side view, realistic irregular white pied patches on body and shoulder,
consistent body size, complete long thin tail with tuft,
pixel-art sprite, readable silhouette at Windows taskbar size,
no text, no border, no scenery, no cast shadow, no multiple animals,
no cropped body parts, no costume, no human-like pose
```

## Review Criteria

- Silhouette remains recognizable on light and dark backgrounds.
- Contact point does not jump sharply between frames.
- Body size does not unnaturally inflate or shrink during motion.
- Action frames are distinct poses, not simple recolors or vertical offsets.
- Different species are distinguishable even in a single-color silhouette.
- Pied patch patterns remain visible after downscaling to `96x64`.
- Coat colors are checked against real degu reference images by approximate RGB direction, not only by color names.
