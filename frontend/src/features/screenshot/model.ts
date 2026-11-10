export type Rect = { x: number; y: number; width: number; height: number }
export type Tool = 'select' | 'rectangle' | 'ellipse' | 'arrow' | 'text'
export type Annotation = Rect & {
  id: number
  type: Exclude<Tool, 'select'>
  color: string
  stroke: number
  text?: string
  fontSize: number
  endX?: number
  endY?: number
}
export type Frame = {
  image: string
  width: number
  height: number
  added?: boolean
  workArea?: Rect
}
export const colors = [
  '#f43f5e',
  '#f59e0b',
  '#22c55e',
  '#3b82f6',
  '#a855f7',
  '#ffffff',
  '#18181b',
]
export function normalized(x: number, y: number, x2: number, y2: number): Rect {
  return {
    x: Math.min(x, x2),
    y: Math.min(y, y2),
    width: Math.abs(x2 - x),
    height: Math.abs(y2 - y),
  }
}
export function drawAnnotation(ctx: CanvasRenderingContext2D, a: Annotation) {
  ctx.strokeStyle = a.color
  ctx.fillStyle = a.color
  ctx.lineWidth = a.stroke
  ctx.lineCap = 'round'
  ctx.lineJoin = 'round'
  ctx.beginPath()
  if (a.type === 'rectangle') ctx.strokeRect(a.x, a.y, a.width, a.height)
  if (a.type === 'ellipse') {
    ctx.ellipse(
      a.x + a.width / 2,
      a.y + a.height / 2,
      a.width / 2,
      a.height / 2,
      0,
      0,
      Math.PI * 2,
    )
    ctx.stroke()
  }
  if (a.type === 'text') {
    ctx.font = `500 ${a.fontSize}px "Segoe UI", sans-serif`
    ctx.textBaseline = 'top'
    ;(a.text || '')
      .split('\n')
      .forEach((line, i) =>
        ctx.fillText(line, a.x, a.y + i * a.fontSize * 1.35),
      )
  }
  if (a.type === 'arrow') {
    const ex = a.endX ?? a.x + a.width,
      ey = a.endY ?? a.y + a.height,
      angle = Math.atan2(ey - a.y, ex - a.x),
      head = 12 + a.stroke * 2
    ctx.moveTo(a.x, a.y)
    ctx.lineTo(ex, ey)
    ctx.stroke()
    ctx.beginPath()
    ctx.moveTo(ex, ey)
    ctx.lineTo(
      ex - head * Math.cos(angle - Math.PI / 6),
      ey - head * Math.sin(angle - Math.PI / 6),
    )
    ctx.lineTo(
      ex - head * Math.cos(angle + Math.PI / 6),
      ey - head * Math.sin(angle + Math.PI / 6),
    )
    ctx.closePath()
    ctx.fill()
  }
}
export async function exportScreenshot(
  frame: Frame,
  selection: Rect,
  annotations: Annotation[],
) {
  const image = new Image()
  image.src = frame.image
  await image.decode()
  const canvas = document.createElement('canvas')
  canvas.width = Math.round(selection.width)
  canvas.height = Math.round(selection.height)
  const ctx = canvas.getContext('2d')!
  ctx.translate(-selection.x, -selection.y)
  ctx.drawImage(image, 0, 0)
  annotations.forEach((a) => drawAnnotation(ctx, a))
  return canvas.toDataURL('image/png')
}
