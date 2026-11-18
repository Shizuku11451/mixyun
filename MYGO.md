# MyGo: what MixMusic misses

What [MyGo](https://github.com/egoist/mygo) lacks, gets wrong, or could do
for less, as MixMusic meets it: each entry a candidate PR to MyGo. Add one
as you find it (see [AGENTS.md](AGENTS.md)); strike it when MyGo fixes it.

Each entry: the MyGo version, where (file and function), what happens, how
it shows in MixMusic, the workaround, and its status.

## Missing

None yet.

## Bugs

None yet.

## Could cost less

### A blurred image is blurred again every frame (Metal)

- **Version:** v0.8.1
- **Where:** `internal/gpu/metal/metal.go`, `Renderer.blurImage`
- **What:** an image drawn with `Blur` is downsampled and blurred into a
  texture of the frame's own on every frame, even when the image and the
  blur are the same as the frame before.
- **In MixMusic:** the now playing page's cover backdrop (`coverBackdrop`,
  a 96 px cover blurred by 60) is blurred again each frame while the
  visualizer or the lyrics move.
- **Workaround:** none needed: from 96 px, the cost is small. Not measured.
- **Status:** not reported.
