---
date: 2026-11-20
status: accepted
---
# Native renderer: move to OpenGL 3.3 core before 3D characters, Vulkan deferred

**Context:** Backlog item 027 renders OpenKakutou-exclusive 3D characters (roadmap decision `028`: glTF 2.0, skinned animation, `KHR_materials_variants` palettes) in the native build. The native renderer (`native/internal/gfx/`) is OpenGL **2.1 fixed-function**: `gl.Ortho`/`gl.MatrixMode` matrices, client-side vertex arrays (`gl.VertexPointer`, `gl.ColorPointer`), no shader (`gfx.go:54-78`, `gfx.go:142-174`, `uilist.go:232-236`). Skinned meshes need vertex shaders (joint matrices per vertex); fixed-function GL would mean skinning every vertex on the CPU each frame, against this repo's frame-budget and no-per-frame-allocation constraints.

The Product Owner asked whether to adopt Vulkan. Facts checked in the Ikemen GO reference checkout (`~/.cache/ikemen-go-ref`, commit `54d4148a`, 2026-11-17):
- Ikemen GO ships three renderers selected by `Video.RenderMode`: OpenGL 3.3 (`src/render_gl33.go`, 2,569 lines), OpenGL ES 3.2 (`src/render_gles32.go`, 2,515 lines) and Vulkan 1.3 (`src/render_vk.go` + `src/render_vk_swap.go`, 11,466 lines), Vulkan through `github.com/Eiton/vulkan`.
- iOS uses Vulkan only (`src/util_ios.go:138`, via MoltenVK); Android offers OpenGL ES 3.2 or Vulkan 1.3 (`src/util_android.go:150-161`).
- The Vulkan renderer draws 3D models (embedded `shaders/model.vert.spv`/`model.frag.spv`, `render_vk.go:29-30`).

Reference low-end machine (the Product Owner's): AMD Ryzen 7 3750H with integrated Radeon Vega 10 Mobile, 2 GiB carved-out VRAM (`mem_info_vram_total`), Mesa 26.1.6. `vulkaninfo --summary` (2026-11-20): RADV `AMD Radeon Vega 10 Graphics (RADV RAVEN)`, device `apiVersion` 1.4.354, conformance 1.4.0.0, but Vulkan loader (instance) 1.3.280 — so Vulkan 1.3 is what an application gets there, the version Ikemen GO targets.

**Decision:**
1. **Move `gfx` to OpenGL 3.3 core profile** (shaders, VAO/VBO, matrices computed in Go) as its own item (071), before item 027. Same version as Ikemen GO's desktop renderer; runs on Windows, Linux and macOS (core profile with forward compatibility).
2. **Keep go-gl confined to `internal/gfx`**, as today (`cmd/openkakutou/main.go` is the only importer; game code describes, `gfx` draws). The 3D character renderer of item 027 lives there too. That package's API is the seam a second backend would plug into; **no `Renderer` interface is introduced while there is a single backend**.
3. **Vulkan is deferred** (item 072, blocked), with explicit triggers: Android rendering is scheduled (roadmap backlog `014`), an Apple target needs it (OpenGL deprecated on macOS, absent on iOS), or a measured frame-budget overrun on the reference machine that profiling attributes to the OpenGL driver rather than to GPU shading or memory.
4. **Frame-budget target for 3D**: 60 fps held (no frame over 16.6 ms) at 1080p on the reference machine, in a match with two 3D characters. The per-character asset budget (triangles, texture size) is set and measured in item 027.

**Reason:** Without item 071, item 027 cannot render skinned meshes on the GPU at all. A match is two characters and a stage: few draw calls, so the Vega 10 is limited by shading and shared memory bandwidth, not by OpenGL driver overhead — the cost Vulkan removes. Vulkan would cost about four times Ikemen GO's OpenGL renderer in code for no measured gain on the targets shipped today (Windows, macOS, Linux). Confining go-gl to one package keeps a later Vulkan backend an addition to `gfx`, not a rewrite of the game.

**Rejected alternatives:**
- *Vulkan now, OpenGL dropped* — four times the rendering code, macOS needs MoltenVK, no gain on the shipped desktop targets; revisited when a trigger of point 3 fires.
- *OpenGL and Vulkan both, behind a `Renderer` interface now* — two backends to keep at parity before the second is needed; the `gfx` package boundary already isolates go-gl.
- *Keep OpenGL 2.1, skin on the CPU* — per-vertex skinning each frame for two characters eats the frame budget on an integrated GPU's CPU side and allocates or rewrites vertex buffers every frame.
- *OpenGL ES 3.0 everywhere (one dialect for desktop and Android)* — would pre-solve Android, still deferred (roadmap decision `027`); desktop GL 3.3 matches Ikemen GO and the Android choice stays with roadmap backlog `014`.
