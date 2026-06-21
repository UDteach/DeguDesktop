# Coat RGB Audit

Date: 2026-06-20

This audit checks the release sprites against public degu color references. The reference photos have mixed lighting, bedding, and backgrounds, so the RGB values are approximate medians from non-background coat pixels rather than biological constants.

Reference pages:

- Degutopia, "What Colours Do Degus Come In?": https://www.degutopia.co.uk/degucolours.htm
- Atlantis Rattery, Degu / Degutopia gallery examples: https://www.atlantisrattery.com/degu.html
- Dein! Degus, Farbschlage reference examples: https://www.dein-degu.de/degus-farbschlage/

## Findings

| Coat | Reference direction | Sprite sample | Result |
|---|---|---:|---|
| Wild agouti | brown/ticked agouti | `#775738` median | kept warm brown, not gray |
| Blue | low-saturation slate / greige, not vivid blue | `#585A54` median | saturated blue was removed |
| White / cream | warm cream with pale highlights | `#C2AC8C` median, 42.2% light patch sample | reads as cream rather than pure white |
| Sand / champagne | warm tan/champagne | `#9D7A4D` median | kept tan/champagne, not orange |
| Black pied | black base with irregular white patches | 25.9% light patch sample | ImageGen guide patch map, not oval mask |
| Agouti pied | agouti base with irregular white patches | 22.2% light patch sample | ImageGen guide patch map, not oval mask |
| Blue pied | slate base with irregular white patches | `#766C5C` median, 19.1% light patch sample | gray/taupe pied, not saturated blue |
| Cream pied | cream base with irregular white patches | `#C9B598` median, 56.3% light patch sample | fixed from low-contrast recolor to visible cream pied |

## Implementation Notes

- Runtime sprites stay on a fixed `96x64` canvas for every frame.
- The base `wild_agouti` motion set is 62 ImageGen frame PNGs.
- Pied variants use ImageGen coat-guide files under `assets/source/coat-guides`.
- The importer transfers each guide's irregular light patch map across every motion frame, then applies the coat palette.
- Contact-sheet QA is local-only under `.codex/qa` and is not used as runtime source material.
