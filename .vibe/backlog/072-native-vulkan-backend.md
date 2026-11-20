---
status: blocked
depends_on: [071, 027]
---
# Vulkan backend for the native renderer

## Description
Decision 028 defers Vulkan. Ikemen GO already ships a Vulkan 1.3 renderer next to OpenGL 3.3 and OpenGL ES 3.2 (`src/render_vk.go`, through `github.com/Eiton/vulkan`; the only renderer on iOS). Add a Vulkan backend inside `internal/gfx`, chosen by a `video.json` setting with OpenGL as the fallback, drawing the same 2D screens and 3D characters as the OpenGL backend.

**Blocked until one trigger fires** (decision 028, point 3):
- Android rendering is scheduled (roadmap backlog `014`);
- an Apple target needs it (OpenGL deprecated on macOS, absent on iOS);
- a measured frame-budget overrun on the reference machine (Radeon Vega 10 Mobile) that profiling attributes to the OpenGL driver, not to GPU shading or memory.

When unblocked, start by deciding whether a `Renderer` interface is introduced inside `gfx` and how both backends are kept at parity.

## Acceptance Criteria
- [ ] The trigger that unblocked this item is named in a decision
- [ ] A Vulkan backend renders every screen and a match, 2D and 3D characters, with the same `-capture` screenshots as OpenGL (within the 071 tolerance)
- [ ] `video.json` selects the backend; an unavailable Vulkan falls back to OpenGL with a notice
- [ ] The frame-budget benchmark (item 025) runs on both backends
