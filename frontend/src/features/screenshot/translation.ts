import type { Rect } from './model'

export type EraseRun = { x: number; y: number; width: number; color: number[] }
export type OCRRegion = Rect & {
  erase?: EraseRun[]
  text: string
  score: number
  boxes?: Rect[]
}

// Center actual glyph bounds, reserving a rasterization margin where space permits.
export function verticalTextLayout(
  originalHeight: number,
  inkHeight: number,
  availableHeight: number,
) {
  const height = Math.min(
    availableHeight,
    Math.max(originalHeight, inkHeight + 2),
  )
  return { height, offset: Math.max(0, (height - inkHeight) / 2) }
}

// Estimate one font size per translation group; never shrink it to fit a result.
export function paintTranslation(
  ctx: CanvasRenderingContext2D,
  region: OCRRegion,
  text: string,
  options?: { bounds: Rect; obstacles: Rect[] },
) {
  const rawSource = region.boxes?.length
    ? region.boxes.map((box) => ({
        ...box,
        x: region.x + box.x,
        y: region.y + box.y,
      }))
    : [region]
  const source = rawSource.map((box) => ({
    ...box,
    x: Math.round(box.x),
    y: Math.round(box.y),
    width: Math.round(box.width),
    height: Math.round(box.height),
  }))
  // Space is measured after the recognized source glyphs are removed, not on raw ink.
  const cleanupRows = new Map<
    number,
    { left: number; right: number; color: number[] }[]
  >()
  for (const run of region.erase ?? []) {
    const y = Math.round(region.y + run.y)
    const row = cleanupRows.get(y) ?? []
    row.push({
      left: Math.round(region.x + run.x),
      right: Math.round(region.x + run.x + run.width),
      color: run.color,
    })
    cleanupRows.set(y, row)
  }
  const willBeCleared = (x: number, y: number) =>
    (cleanupRows.get(Math.round(y)) ?? []).some(
      (run) => x >= run.left && x < run.right,
    )
  const heights = source.map((box) => box.height).sort((a, b) => a - b)
  const glyphHeight = heights[Math.floor((heights.length - 1) / 2)] * 0.85
  ctx.save()
  ctx.font = '500 100px "Segoe UI", sans-serif'
  const measured = ctx.measureText('汉Ag')
  const glyphRatio =
    ((measured.actualBoundingBoxAscent ?? 80) +
      (measured.actualBoundingBoxDescent ?? 20)) /
    100
  const size = Math.max(6, Math.round(glyphHeight / glyphRatio))
  ctx.font = `500 ${size}px "Segoe UI", sans-serif`
  ctx.textBaseline = 'alphabetic'
  const targetMetrics = ctx.measureText(text)
  const targetAscent = targetMetrics.actualBoundingBoxAscent || size * 0.8
  const inkHeight = Math.max(
    1,
    targetAscent + (targetMetrics.actualBoundingBoxDescent || 0),
  )
  const lineHeight = inkHeight + Math.max(2, Math.round(size * 0.12))

  const chars = Array.from(text.replace(/\r/g, ''))
  let next = 0
  const placements: { text: string; x: number; y: number }[] = []
  const fills = source.map((box) => {
    let right = options
      ? options.bounds.x + options.bounds.width
      : box.x + box.width
    for (const obstacle of [...(options?.obstacles ?? []), ...source]) {
      if (
        obstacle.x >= box.x + box.width - 1 &&
        obstacle.y < box.y + box.height &&
        obstacle.y + obstacle.height > box.y
      )
        right = Math.min(right, obstacle.x - 2)
    }
    // Expansion is permitted only into blank space; original OCR slots are never vetoed.
    const localRun = (region.erase ?? []).find(
      (run) =>
        run.y + region.y >= box.y &&
        run.y + region.y < box.y + box.height &&
        run.x + region.x + run.width > box.x &&
        run.x + region.x < box.x + box.width,
    )
    const background =
      localRun?.color ??
      Array.from(ctx.getImageData(box.x, box.y, 1, 1).data).slice(0, 3)
    const extraWidth = Math.max(0, Math.floor(right - box.x - box.width))
    if (extraWidth > 0) {
      const strip = ctx.getImageData(
        box.x + box.width,
        box.y,
        extraWidth,
        box.height,
      ).data
      outer: for (let x = 0; x < extraWidth; x++)
        for (let y = 0; y < box.height; y++) {
          const i = (y * extraWidth + x) * 4
          if (
            !willBeCleared(box.x + box.width + x, box.y + y) &&
            background.some((c, j) => Math.abs(strip[i + j] - c) > 32)
          ) {
            right = box.x + box.width + x
            break outer
          }
        }
    }
    let left = box.x
    // Right-aligned short labels (Ad, buttons, badges) can borrow blank space on the left.
    if (options && source.length === 1 && box.width < size * 3) {
      const desired = Math.max(
        0,
        ctx.measureText(text.replace(/\s+/g, ' ')).width - (right - box.x),
      )
      let candidate = Math.max(
        Math.ceil(options.bounds.x),
        box.x - Math.ceil(Math.min(size * 2, desired)),
      )
      for (const obstacle of options.obstacles) {
        if (
          obstacle.x + obstacle.width <= box.x &&
          obstacle.y < box.y + box.height &&
          obstacle.y + obstacle.height > box.y
        )
          candidate = Math.max(
            candidate,
            Math.ceil(obstacle.x + obstacle.width + 2),
          )
      }
      const extra = box.x - candidate
      if (extra > 0) {
        const pixels = ctx.getImageData(
          candidate,
          box.y,
          extra,
          box.height,
        ).data
        left = candidate
        outerLeft: for (let x = extra - 1; x >= 0; x--)
          for (let y = 0; y < box.height; y++) {
            const i = (y * extra + x) * 4
            if (
              !willBeCleared(candidate + x, box.y + y) &&
              background.some((c, j) => Math.abs(pixels[i + j] - c) > 32)
            ) {
              left = candidate + x + 1
              break outerLeft
            }
          }
      }
    }
    const availableWidth = Math.max(box.width, Math.floor(right - left))
    let bottom = options
      ? options.bounds.y + options.bounds.height
      : box.y + box.height
    for (const obstacle of options?.obstacles ?? []) {
      if (
        obstacle.y >= box.y + box.height - 1 &&
        obstacle.x < box.x + box.width &&
        obstacle.x + obstacle.width > box.x
      )
        bottom = Math.min(bottom, obstacle.y - 2)
    }
    let rows = Math.max(
      0,
      Math.floor((bottom - box.y - inkHeight) / lineHeight) + 1,
    )
    let used = 0
    const firstPlacement = placements.length
    while (rows-- > 0 && next < chars.length) {
      let line = ''
      const remaining = chars.slice(next).join('').split('\n')[0]
      // Keep short labels on one line with at most 15% horizontal condensation.
      if (
        remaining &&
        ctx.measureText(remaining).width <= availableWidth / 0.85
      ) {
        line = remaining
        next += Array.from(remaining).length
        if (chars[next] === '\n') next++
      }
      let wrapping = !line
      while (wrapping && next < chars.length) {
        const char = chars[next]
        if (char === '\n') {
          next++
          break
        }
        if (ctx.measureText(line + char).width > availableWidth) break
        line += char
        next++
      }
      placements.push({ text: line, x: left, y: box.y + used * lineHeight })
      used++
    }
    const contentHeight = used > 0 ? (used - 1) * lineHeight + inkHeight : 0
    const centered = verticalTextLayout(
      box.height,
      contentHeight,
      Math.max(box.height, bottom - box.y),
    )
    for (let i = firstPlacement; i < placements.length; i++)
      placements[i].y += centered.offset
    return {
      ...box,
      x: left,
      width: availableWidth,
      height: centered.height,
      sample: ctx.getImageData(
        Math.max(0, Math.floor(box.x)),
        Math.max(0, Math.floor(box.y)),
        1,
        1,
      ).data,
    }
  })
  let fits = next === chars.length
  let reason = fits ? 'success' : 'insufficient_space'
  if (fits) {
    const left = Math.floor(
      Math.min(
        ...fills.map((box) => box.x),
        ...(region.erase ?? []).map((run) => run.x + region.x),
      ),
    )
    const top = Math.floor(
      Math.min(
        ...fills.map((box) => box.y),
        ...(region.erase ?? []).map((run) => run.y + region.y),
      ),
    )
    const width =
      Math.ceil(
        Math.max(
          ...fills.map((box) => box.x + box.width),
          ...(region.erase ?? []).map((run) => run.x + region.x + run.width),
        ),
      ) - left
    const height =
      Math.ceil(
        Math.max(
          ...fills.map((box) => box.y + box.height),
          ...(region.erase ?? []).map((run) => run.y + region.y + 1),
        ),
      ) - top
    const original = ctx.getImageData(left, top, width, height)
    const layer = document.createElement('canvas')
    layer.width = width
    layer.height = height
    const ink = layer.getContext('2d')!
    ink.font = ctx.font
    ink.textBaseline = 'alphabetic'
    const bg =
      region.erase?.[0]?.color ?? Array.from(fills[0].sample).slice(0, 3)
    ink.fillStyle =
      bg[0] * 0.299 + bg[1] * 0.587 + bg[2] * 0.114 > 140
        ? '#18181b'
        : '#ffffff'
    placements.forEach((line) => {
      const slot = fills.find(
        (box) =>
          line.x === box.x && line.y >= box.y && line.y < box.y + box.height,
      )!
      ink.save()
      ink.beginPath()
      ink.rect(slot.x - left, slot.y - top, slot.width, slot.height)
      ink.clip()
      ink.fillText(
        line.text,
        line.x - left,
        line.y - top + targetAscent,
        slot.width,
      )
      ink.restore()
    })
    const runs = (region.erase ?? []).map((run) => ({
      ...run,
      x: run.x + region.x - left,
      y: run.y + region.y - top,
    }))
    const output = mergeTextPixels(
      original.data,
      ink.getImageData(0, 0, width, height).data,
      width,
      runs,
      bg,
    )
    original.data.set(output)
    ctx.putImageData(original, left, top)
  }

  ctx.restore()
  return {
    fits,
    reason,
    fontSize: size,
    inkHeight,
    lineHeight,
    lines: placements.length,
    availableWidth: Math.max(...fills.map((box) => box.width)),
    textCharacters: chars.length,
    placedCharacters: next,
  }
}

// Clear only OCR text pixels, then draw approved glyphs without a per-pixel veto.
export function mergeTextPixels(
  original: Uint8ClampedArray,
  glyphs: Uint8ClampedArray,
  width: number,
  runs: EraseRun[],
  _background?: number[],
) {
  const output = new Uint8ClampedArray(original)
  for (const run of runs)
    for (let x = Math.round(run.x); x < Math.round(run.x + run.width); x++) {
      const i = (Math.round(run.y) * width + x) * 4
      if (x < 0 || x >= width || i < 0 || i + 3 >= output.length) continue
      for (let c = 0; c < 3; c++) output[i + c] = run.color[c]
    }
  for (let i = 0; i < glyphs.length; i += 4) {
    const alpha = glyphs[i + 3] / 255
    if (!alpha) continue
    for (let c = 0; c < 3; c++)
      output[i + c] = Math.round(
        output[i + c] * (1 - alpha) + glyphs[i + c] * alpha,
      )
  }
  return output
}

export async function translateRegions(
  regions: OCRRegion[],
  translate: (region: OCRRegion) => Promise<string>,
  completed: (region: OCRRegion, text: string) => void,
  failed: () => void,
) {
  let next = 0
  await Promise.all(
    Array.from({ length: Math.min(4, regions.length) }, async () => {
      while (next < regions.length) {
        const region = regions[next++]
        try {
          const text = await translate(region)
          if (!text.trim()) throw new Error('Empty translation')
          completed(region, text)
        } catch {
          failed()
        }
      }
    }),
  )
}
