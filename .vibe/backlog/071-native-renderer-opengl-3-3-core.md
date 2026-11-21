---
status: todo
depends_on: []
---
# Native renderer on OpenGL 3.3 core

## Description
Decision 028. `native/internal/gfx` draws with OpenGL 2.1 fixed-function (`gl.Ortho`, `gl.MatrixMode`, client-side `gl.VertexPointer`/`gl.ColorPointer`, no shader). Item 027 needs vertex shaders to skin 3D characters on the GPU. Port `gfx` to an OpenGL 3.3 core profile context: shader programs for colored quads, textured sprites and bitmap text, VAO/VBO buffers reused across frames, projection matrices computed in Go. go-gl stays imported by `internal/gfx` only.

🔒 MUST: no visible change — every screen and a match render the same pixels as before.

## Acceptance Criteria
- [ ] The SDL context requests OpenGL 3.3 core (forward-compatible, so it also opens on macOS); no fixed-function call remains (`gl.Ortho`, `gl.MatrixMode`, `gl.LoadIdentity`, `gl.EnableClientState`, `gl.VertexPointer`, `gl.ColorPointer`, `gl.TexCoordPointer`)
- [ ] `-frames N -capture out.png` screenshots of the title, select, versus and match screens match the OpenGL 2.1 build's (pixel diff within a documented tolerance), in the same test setup as today's captures
- [ ] Still one draw call per batch and no per-frame heap allocation in the render loop
- [ ] Startup on a machine without OpenGL 3.3 fails with a clear message naming the required version
- [ ] `.vibe/modules` and `CLAUDE.md` name OpenGL 3.3 core instead of 2.1
- [ ] Textured sprites are batched by texture and blend state like colored quads (audit 2026-11-21: `SpriteSized` and its siblings flush and bind per call, and a tiled stage layer draws one sprite per tile each frame); the three copies of the textured-quad code become one
