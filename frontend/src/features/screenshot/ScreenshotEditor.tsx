import {
  useEffect,
  useLayoutEffect,
  useRef,
  useState,
  type PointerEvent,
} from 'react'
import {
  ArrowUpRight,
  Circle,
  Download,
  Copy,
  Languages,
  Pin,
  RectangleHorizontal,
  ScanLine,
  Type,
  Undo2,
  Redo2,
  X,
  Crop,
} from 'lucide-react'
import {
  CancelScreenshot,
  PinScreenshot,
  SaveScreenshot,
  CopyScreenshot,
  ScreenshotReady,
  ScrollScreenshot,
} from '../../../wailsjs/go/app/WindowService'
import {
  colors,
  normalized,
  exportScreenshot,
  type Annotation,
  type Frame,
  type Rect,
  type Tool,
} from './model'
import './screenshot.css'

type Drag = {
  startX: number
  startY: number
  rect: Rect
  kind: 'new' | 'move' | 'resize' | 'draw'
  corner?: string
  id?: number
  before: EditSnapshot
  started?: boolean
}
type EditSnapshot = {
  selection: Rect | null
  annotations: Annotation[]
  ready: boolean
}
const tools = [
  { id: 'rectangle', label: '矩形', icon: RectangleHorizontal },
  { id: 'ellipse', label: '圆形', icon: Circle },
  { id: 'arrow', label: '箭头', icon: ArrowUpRight },
  { id: 'text', label: '文字', icon: Type },
] as const
export function ScreenshotEditor({ initial }: { initial: Frame }) {
  const [frame, setFrame] = useState(initial),
    [selection, setSelection] = useState<Rect | null>(
      initial.workArea ?? {
        x: 0,
        y: 0,
        width: initial.width,
        height: initial.height,
      },
    ),
    [ready, setReady] = useState(false)
  const [tool, setTool] = useState<Tool>('select'),
    [color, setColor] = useState(colors[3]),
    [stroke, setStroke] = useState(4),
    [fontSize, setFontSize] = useState(26)
  const [annotations, setAnnotations] = useState<Annotation[]>([]),
    [selected, setSelected] = useState<number | null>(null)
  const [text, setText] = useState<{
      x: number
      y: number
      value: string
    } | null>(null),
    [busy, setBusy] = useState(false),
    [message, setMessage] = useState('')
  const [pinned, setPinned] = useState<string | null>(null),
    [viewport, setViewport] = useState({
      width: innerWidth,
      height: innerHeight,
    })
  const [scrollRegion, setScrollRegion] = useState<Rect | null>(null)
  const stage = useRef<HTMLDivElement>(null),
    drag = useRef<Drag | null>(null),
    nextId = useRef(1)
  const scale = Math.min(
    viewport.width / frame.width,
    viewport.height / frame.height,
  )
  const active = annotations.find((a) => a.id === selected)
  const [cropSelected, setCropSelected] = useState(false)
  const [interacting, setInteracting] = useState(false)
  const dock = useRef<HTMLDivElement>(null)
  const [dockSize, setDockSize] = useState({ width: 470, height: 50 })
  const preferences = useRef<
    Record<string, { color: string; stroke: number; fontSize: number }>
  >({})
  const optionTool = active?.type ?? (tool === 'select' ? null : tool)
  const bound = active
    ? active.type === 'rectangle' || active.type === 'ellipse'
      ? active
      : null
    : cropSelected
      ? selection
      : null
  const exporting = useRef(false)
  const textInput = useRef<HTMLTextAreaElement>(null)
  const textWidth = (() => {
    if (!text || !selection) return 0
    const ctx = document.createElement('canvas').getContext('2d')!
    ctx.font = `500 ${fontSize}px "Segoe UI", sans-serif`
    return Math.max(
      fontSize * 2,
      Math.min(
        selection.x + selection.width - text.x,
        Math.max(
          fontSize * 7,
          ...text.value
            .split('\n')
            .map((line) => ctx.measureText(line).width + 4 / scale),
        ),
      ),
    )
  })()

  const snapshot = useRef<EditSnapshot>({ selection, annotations, ready })
  snapshot.current = { selection, annotations, ready }
  const history = useRef<{ past: EditSnapshot[]; future: EditSnapshot[] }>({
    past: [],
    future: [],
  })
  const [, refreshHistory] = useState(0)
  function remember(before = snapshot.current) {
    history.current.past = [...history.current.past.slice(-49), before]
    history.current.future = []
    refreshHistory((n) => n + 1)
  }
  function restore(edit: EditSnapshot) {
    setSelection(edit.selection)
    setAnnotations(edit.annotations)
    setReady(edit.ready)
    setSelected((current) =>
      edit.annotations.some((a) => a.id === current) ? current : null,
    )
    setText(null)
    setCropSelected(false)
  }
  function undo() {
    const previous = history.current.past.pop()
    if (!previous) return
    history.current.future.push(snapshot.current)
    restore(previous)
    refreshHistory((n) => n + 1)
  }
  function redo() {
    const next = history.current.future.pop()
    if (!next) return
    history.current.past.push(snapshot.current)
    restore(next)
    refreshHistory((n) => n + 1)
  }
  useLayoutEffect(() => {
    const element = dock.current
    if (!element) return
    const observer = new ResizeObserver(() =>
      setDockSize({ width: element.offsetWidth, height: element.offsetHeight }),
    )
    observer.observe(element)
    return () => observer.disconnect()
  }, [ready, pinned])
  useLayoutEffect(() => {
    const input = textInput.current
    if (!input) return
    input.style.height = 'auto'
    input.style.height = `${Math.max(fontSize * scale * 1.5, input.scrollHeight + 2)}px`
  }, [text, fontSize, scale])
  useEffect(() => {
    if (text) textInput.current?.focus({ preventScroll: true })
  }, [text !== null])
  useEffect(() => {
    let raf = requestAnimationFrame(() => {
      raf = requestAnimationFrame(() => void ScreenshotReady())
    })
    const resize = () => setViewport({ width: innerWidth, height: innerHeight })
    addEventListener('resize', resize)
    return () => {
      cancelAnimationFrame(raf)
      removeEventListener('resize', resize)
    }
  }, [])
  useEffect(() => {
    const key = (e: KeyboardEvent) => {
      if (busy) return
      if (e.key === 'Escape') {
        e.preventDefault()
        if (pinned) void CancelScreenshot()
        else if (text) setText(null)
        else if (tool !== 'select') {
          setTool('select')
          setSelected(null)
          setCropSelected(false)
        } else void CancelScreenshot()
        return
      }
      if (
        e.target instanceof HTMLTextAreaElement ||
        e.target instanceof HTMLInputElement
      )
        return
      if (e.key === 'Enter' && !e.isComposing && !pinned) {
        e.preventDefault()
        void output('copy')
        return
      }
      if (
        (e.ctrlKey || e.metaKey) &&
        ['z', 'y'].includes(e.key.toLowerCase())
      ) {
        e.preventDefault()
        if (e.key.toLowerCase() === 'y' || e.shiftKey) redo()
        else undo()
      }
      if (e.key === 'Delete' && selected !== null) {
        remember()
        setAnnotations((a) => a.filter((x) => x.id !== selected))
        setSelected(null)
      }
    }
    addEventListener('keydown', key)
    return () => removeEventListener('keydown', key)
  }, [selection, selected, busy, tool, text, pinned, frame, annotations])
  function point(e: PointerEvent) {
    const r = stage.current!.getBoundingClientRect()
    return {
      x: Math.max(0, Math.min(frame.width, (e.clientX - r.left) / scale)),
      y: Math.max(0, Math.min(frame.height, (e.clientY - r.top) / scale)),
    }
  }
  function updateRect(r: Rect, id?: number) {
    if (id !== undefined)
      setAnnotations((a) => a.map((x) => (x.id === id ? { ...x, ...r } : x)))
    else setSelection(r)
  }
  function toolSettings(id: Tool) {
    return (
      preferences.current[id] ?? {
        color: colors[3],
        stroke: id === 'arrow' ? 2 : 4,
        fontSize: 26,
      }
    )
  }
  function restoreToolSettings() {
    const settings = toolSettings(tool)
    setColor(settings.color)
    setStroke(settings.stroke)
    setFontSize(settings.fontSize)
    return settings
  }
  function down(e: PointerEvent<HTMLDivElement>) {
    if (e.button !== 0 || busy || pinned || text) return
    setMessage('')
    const p = point(e)
    const target = e.target as HTMLElement
    const corner = target.dataset.corner
    const inside =
      selection &&
      p.x >= selection.x &&
      p.x <= selection.x + selection.width &&
      p.y >= selection.y &&
      p.y <= selection.y + selection.height
    const before = snapshot.current
    const annotation = target.closest('[data-annotation]') as HTMLElement | null
    const id = Number(annotation?.dataset.annotation)
    if (ready && annotation && Number.isFinite(id) && !e.altKey) {
      e.currentTarget.setPointerCapture(e.pointerId)
      setInteracting(true)
      const a = annotations.find((a) => a.id === id)!
      setSelected(id)
      setCropSelected(false)
      setColor(a.color)
      setStroke(a.stroke)
      setFontSize(a.fontSize)
      drag.current = {
        startX: p.x,
        startY: p.y,
        rect: { ...a },
        kind: 'move',
        id,
        before,
      }
      return
    }
    if (ready && tool === 'text' && inside && !corner) {
      restoreToolSettings()
      e.preventDefault()
      setSelected(null)
      setCropSelected(false)
      setText({ x: p.x, y: p.y, value: '' })
      return
    }
    e.currentTarget.setPointerCapture(e.pointerId)
    setInteracting(true)
    if (corner && bound) {
      drag.current = {
        startX: p.x,
        startY: p.y,
        rect: { ...bound },
        kind: 'resize',
        corner,
        id: active?.id,
        before,
      }
      return
    }
    if (!ready || (tool === 'select' && !inside)) {
      setCropSelected(false)
      setSelected(null)
      setInteracting(false)
      drag.current = {
        startX: p.x,
        startY: p.y,
        rect: { x: p.x, y: p.y, width: 0, height: 0 },
        kind: 'new',
        before,
      }
      return
    }
    if (tool === 'select' && inside) {
      setCropSelected(true)
      setSelected(null)
      drag.current = {
        startX: p.x,
        startY: p.y,
        rect: { ...selection! },
        kind: 'move',
        before,
      }
      return
    }
    if (!inside || tool === 'select' || tool === 'text') return
    const settings = restoreToolSettings()
    const a: Annotation = {
      id: nextId.current++,
      type: tool,
      x: p.x,
      y: p.y,
      width: 0,
      height: 0,
      ...settings,
      endX: p.x,
      endY: p.y,
    }
    setAnnotations((list) => [...list, a])
    setCropSelected(false)
    setSelected(null)
    drag.current = {
      startX: p.x,
      startY: p.y,
      rect: a,
      kind: 'draw',
      id: a.id,
      before,
    }
  }
  function up() {
    const d = drag.current
    drag.current = null
    setInteracting(false)
    if (!d) return
    if (d.kind === 'new') {
      setCropSelected(false)
      if (!selection || selection.width < 8 || selection.height < 8)
        setSelection(d.before.selection)
      setReady(true)
    } else if (d.id === undefined) setReady(true)
    if (d.kind === 'draw') setSelected(null)
    if (d.id !== undefined && d.kind === 'draw') {
      const a = snapshot.current.annotations.find((a) => a.id === d.id)
      if (a && a.width < 2 && a.height < 2) {
        setAnnotations(d.before.annotations)
        setSelected(null)
        return
      }
    }
    if (
      JSON.stringify(d.before) !== JSON.stringify(snapshot.current) ||
      !d.before.ready
    )
      remember(d.before)
  }
  function move(e: PointerEvent<HTMLDivElement>) {
    const d = drag.current
    if (!d) return
    const p = point(e)
    if (d.kind === 'new' || d.kind === 'draw') {
      const r = normalized(d.startX, d.startY, p.x, p.y)
      if (d.kind === 'new' && (r.width < 8 || r.height < 8)) return
      if (d.kind === 'new' && !d.started) {
        if (r.width * scale < 4 || r.height * scale < 4) return
        d.started = true
        setInteracting(true)
      }
      if (d.id !== undefined)
        setAnnotations((a) =>
          a.map((x) =>
            x.id === d.id
              ? {
                  ...x,
                  ...r,
                  ...(x.type === 'arrow'
                    ? { x: d.startX, y: d.startY, endX: p.x, endY: p.y }
                    : {}),
                }
              : x,
          ),
        )
      else setSelection(r)
    } else if (d.kind === 'move') {
      const dx = Math.max(
          -d.rect.x,
          Math.min(frame.width - d.rect.x - d.rect.width, p.x - d.startX),
        ),
        dy = Math.max(
          -d.rect.y,
          Math.min(frame.height - d.rect.y - d.rect.height, p.y - d.startY),
        )
      if (d.id !== undefined)
        setAnnotations((a) =>
          a.map((x) =>
            x.id === d.id
              ? {
                  ...x,
                  x: d.rect.x + dx,
                  y: d.rect.y + dy,
                  ...(x.type === 'arrow'
                    ? {
                        endX: (x.endX ?? 0) + (d.rect.x + dx - x.x),
                        endY: (x.endY ?? 0) + (d.rect.y + dy - x.y),
                      }
                    : {}),
                }
              : x,
          ),
        )
      else updateRect({ ...d.rect, x: d.rect.x + dx, y: d.rect.y + dy })
    } else {
      const x = d.corner!.includes('w') ? p.x : d.rect.x,
        y = d.corner!.includes('n') ? p.y : d.rect.y
      const x2 = d.corner!.includes('e') ? p.x : d.rect.x + d.rect.width,
        y2 = d.corner!.includes('s') ? p.y : d.rect.y + d.rect.height
      updateRect(normalized(x, y, x2, y2), d.id)
    }
  }
  function commitText() {
    if (!text?.value.trim()) {
      setText(null)
      return annotations
    }
    const id = nextId.current++
    const canvas = document.createElement('canvas'),
      ctx = canvas.getContext('2d')!
    ctx.font = `500 ${fontSize}px "Segoe UI", sans-serif`
    const maxWidth = textInput.current
      ? textInput.current.clientWidth / scale
      : textWidth
    const lines: string[] = []
    for (const paragraph of text.value.split('\n')) {
      let line = ''
      for (const char of paragraph) {
        if (line && ctx.measureText(line + char).width > maxWidth) {
          lines.push(line)
          line = ''
        }
        line += char
      }
      lines.push(line)
    }
    const next: Annotation[] = [
      ...annotations,
      {
        id,
        type: 'text',
        x: text.x,
        y: text.y,
        width: Math.max(
          fontSize,
          ...lines.map((line) => ctx.measureText(line).width),
        ),
        height: lines.length * fontSize * 1.35,
        color,
        stroke,
        fontSize,
        text: lines.join('\n'),
      },
    ]
    remember()
    setAnnotations(next)
    setSelected(null)
    setCropSelected(false)
    setText(null)
    return next
  }
  function chooseTool(id: Tool) {
    if (text) commitText()
    const next = tool === id ? 'select' : id
    setTool(next)
    setSelected(null)
    setCropSelected(false)
    const settings = toolSettings(next)
    setColor(settings.color)
    setStroke(settings.stroke)
    setFontSize(settings.fontSize)
  }
  function configure(
    patch: Partial<Pick<Annotation, 'color' | 'stroke' | 'fontSize'>>,
  ) {
    if (patch.color !== undefined) setColor(patch.color)
    if (patch.stroke !== undefined) setStroke(patch.stroke)
    if (patch.fontSize !== undefined) setFontSize(patch.fontSize)
    if (optionTool)
      preferences.current[optionTool] = { color, stroke, fontSize, ...patch }
    appearance(patch)
  }
  function appearance(patch: Partial<Annotation>) {
    if (
      selected !== null &&
      active &&
      Object.entries(patch).some(
        ([key, value]) => active[key as keyof Annotation] !== value,
      )
    ) {
      remember()
      setAnnotations((a) =>
        a.map((x) => (x.id === selected ? { ...x, ...patch } : x)),
      )
    }
  }
  async function output(action: 'save' | 'pin' | 'copy' = 'save') {
    if (!selection || exporting.current) return
    exporting.current = true
    const edited = commitText()
    setBusy(true)
    try {
      const image = await exportScreenshot(frame, selection, edited)
      if (action === 'copy') {
        await CopyScreenshot(image)
      } else if (action === 'pin') {
        await PinScreenshot(
          Math.round(selection.width),
          Math.round(selection.height),
        )
        setPinned(image)
      } else {
        const path = await SaveScreenshot(image)
        if (path) setMessage('截图已保存')
      }
    } catch (e) {
      setMessage(String(e))
    } finally {
      exporting.current = false
      setBusy(false)
    }
  }
  async function scroll() {
    if (!selection) return
    setBusy(true)
    setSelected(null)
    setCropSelected(false)
    const r = scrollRegion || selection
    try {
      const result = await ScrollScreenshot(
        Math.round(r.x),
        Math.round(r.y),
        Math.round(r.width),
        Math.round(r.height),
      )
      history.current = { past: [], future: [] }
      refreshHistory((n) => n + 1)
      setScrollRegion(r)
      setFrame(result)
      setSelection({ x: 0, y: 0, width: result.width, height: result.height })
      setAnnotations((list) =>
        scrollRegion
          ? list
          : list.map((a) => ({
              ...a,
              x: a.x - r.x,
              y: a.y - r.y,
              endX: a.endX === undefined ? undefined : a.endX - r.x,
              endY: a.endY === undefined ? undefined : a.endY - r.y,
            })),
      )
      setMessage(
        result.added
          ? '已拼接下一屏，可继续滚动'
          : '没有检测到新的内容：可能已到底部，或选区中没有可滚动内容',
      )
    } catch (e) {
      setMessage(String(e))
    } finally {
      setBusy(false)
    }
  }
  const stageLeft = (viewport.width - frame.width * scale) / 2
  const stageTop = (viewport.height - frame.height * scale) / 2
  const right =
    stageLeft + (selection ? selection.x + selection.width : 0) * scale
  const bottom =
    stageTop + (selection ? selection.y + selection.height : 0) * scale
  const above = stageTop + (selection?.y ?? 0) * scale - dockSize.height - 12
  const dockLeft = Math.max(
    8,
    Math.min(viewport.width - dockSize.width - 8, right - dockSize.width),
  )
  const dockTop = Math.max(
    8,
    Math.min(
      viewport.height - dockSize.height - 8,
      bottom + dockSize.height + 20 <= viewport.height
        ? bottom + 12
        : above >= 8
          ? above
          : bottom - dockSize.height - 12,
    ),
  )
  if (pinned)
    return (
      <div
        className="screenshot-pin"
        style={{ '--wails-draggable': 'drag' } as React.CSSProperties}
      >
        <img src={pinned} draggable={false} alt="置顶截图" />
        <button
          aria-label="关闭置顶截图"
          title="关闭置顶截图 · Esc"
          onClick={() => void CancelScreenshot()}
        >
          <X size={16} />
        </button>
      </div>
    )
  return (
    <div className="screenshot-editor" aria-label="截图编辑器">
      <div
        ref={stage}
        className="screenshot-stage"
        style={{
          width: frame.width * scale,
          height: frame.height * scale,
          left: (viewport.width - frame.width * scale) / 2,
          top: (viewport.height - frame.height * scale) / 2,
          cursor: ready && tool === 'select' ? 'default' : 'crosshair',
        }}
        onPointerDown={down}
        onPointerMove={move}
        onPointerUp={up}
        onPointerCancel={() => {
          if (drag.current) restore(drag.current.before)
          drag.current = null
          setInteracting(false)
        }}
      >
        <img src={frame.image} draggable={false} alt="屏幕截图" />
        <svg
          viewBox={`0 0 ${frame.width} ${frame.height}`}
          className="screenshot-art"
          aria-label="截图标注"
        >
          <defs>
            <mask id="selection-mask">
              <rect width={frame.width} height={frame.height} fill="white" />
              {selection && <rect {...selection} fill="black" />}
            </mask>
            <clipPath id="selection-clip">
              {selection && <rect {...selection} />}
            </clipPath>
          </defs>
          <rect
            width={frame.width}
            height={frame.height}
            fill="rgba(9,9,11,.48)"
            mask="url(#selection-mask)"
          />
          <g clipPath="url(#selection-clip)">
            {annotations.map((a) =>
              a.type === 'text' ? (
                <text
                  key={a.id}
                  data-annotation={a.id}
                  x={a.x}
                  y={a.y}
                  dominantBaseline="text-before-edge"
                  fill={a.color}
                  fontSize={a.fontSize}
                  style={{
                    pointerEvents: 'all',
                    cursor: 'move',
                    fontFamily: '"Segoe UI", sans-serif',
                    fontWeight: 500,
                  }}
                >
                  {(a.text || '').split('\n').map((line, i) => (
                    <tspan key={i} x={a.x} dy={i ? a.fontSize * 1.35 : 0}>
                      {line}
                    </tspan>
                  ))}
                </text>
              ) : a.type === 'arrow' ? (
                <g
                  key={a.id}
                  data-annotation={a.id}
                  stroke={a.color}
                  strokeWidth={a.stroke}
                  fill={a.color}
                  style={{ pointerEvents: 'stroke' }}
                >
                  <defs>
                    <marker
                      id={`arrow-${a.id}`}
                      markerWidth="8"
                      markerHeight="8"
                      refX="7"
                      refY="4"
                      orient="auto"
                    >
                      <path d="M0 0 L8 4 L0 8 Z" stroke="none" />
                    </marker>
                  </defs>
                  <line
                    data-annotation={a.id}
                    x1={a.x}
                    y1={a.y}
                    x2={a.endX}
                    y2={a.endY}
                    markerEnd={`url(#arrow-${a.id})`}
                  />
                </g>
              ) : a.type === 'rectangle' ? (
                <rect
                  key={a.id}
                  data-annotation={a.id}
                  {...{ x: a.x, y: a.y, width: a.width, height: a.height }}
                  stroke={a.color}
                  strokeWidth={a.stroke}
                  fill="transparent"
                  style={{ pointerEvents: 'all' }}
                />
              ) : (
                <ellipse
                  key={a.id}
                  data-annotation={a.id}
                  cx={a.x + a.width / 2}
                  cy={a.y + a.height / 2}
                  rx={a.width / 2}
                  ry={a.height / 2}
                  stroke={a.color}
                  strokeWidth={a.stroke}
                  fill="transparent"
                  style={{ pointerEvents: 'all' }}
                />
              ),
            )}
          </g>
        </svg>
        {selection && (
          <div
            className="screenshot-selection"
            style={{
              left: selection.x * scale,
              top: selection.y * scale,
              width: selection.width * scale,
              height: selection.height * scale,
            }}
          >
            <span
              className="screenshot-size"
              style={{ top: selection.y * scale < 28 ? 4 : -27 }}
            >
              {Math.round(selection.width)} × {Math.round(selection.height)}
            </span>
          </div>
        )}
        {bound &&
          !interacting &&
          !text &&
          !scrollRegion &&
          (['nw', 'n', 'ne', 'e', 'se', 's', 'sw', 'w'] as const).map((c) => (
            <div
              key={c}
              data-corner={c}
              className="screenshot-handle"
              style={{
                left:
                  (bound.x +
                    (c.includes('w')
                      ? 0
                      : c.includes('e')
                        ? bound.width
                        : bound.width / 2)) *
                  scale,
                top:
                  (bound.y +
                    (c.includes('n')
                      ? 0
                      : c.includes('s')
                        ? bound.height
                        : bound.height / 2)) *
                  scale,
                cursor: `${c}-resize`,
              }}
            />
          ))}
        {text && (
          <textarea
            ref={textInput}
            autoFocus
            wrap="soft"
            className="screenshot-text"
            aria-label="截图文字"
            style={{
              left: text.x * scale,
              top: text.y * scale,
              fontSize: fontSize * scale,
              color,
              width: textWidth * scale,
              height:
                Math.max(
                  fontSize * 1.5,
                  text.value.split('\n').length * fontSize * 1.35 + 6,
                ) * scale,
            }}
            value={text.value}
            onChange={(e) => setText({ ...text, value: e.target.value })}
            onPointerDown={(e) => e.stopPropagation()}
            onBlur={commitText}
          />
        )}
      </div>
      {ready && selection && selection.width >= 8 && selection.height >= 8 && (
        <div
          ref={dock}
          className="screenshot-dock"
          style={{
            left: dockLeft,
            top: dockTop,
            visibility: interacting ? 'hidden' : 'visible',
          }}
          onPointerDown={(e) => {
            e.preventDefault()
            e.stopPropagation()
          }}
        >
          <div
            className="screenshot-tools"
            role="toolbar"
            aria-label="截图工具栏"
          >
            {tools.map(({ id, label, icon: Icon }) => (
              <button
                key={id}
                title={label}
                aria-label={label}
                aria-pressed={tool === id}
                disabled={busy}
                className={tool === id ? 'active' : ''}
                onClick={() => chooseTool(id)}
              >
                <Icon size={16} strokeWidth={1.5} />
              </button>
            ))}
            <i />
            <button
              aria-label="撤销"
              title="撤销 · Ctrl+Z"
              disabled={!history.current.past.length || busy}
              onClick={undo}
            >
              <Undo2 size={16} strokeWidth={1.5} />
            </button>
            <button
              aria-label="恢复"
              title="恢复 · Ctrl+Y"
              disabled={!history.current.future.length || busy}
              onClick={redo}
            >
              <Redo2 size={16} strokeWidth={1.5} />
            </button>
            <button
              aria-label="重新选择区域"
              title="重新框选"
              disabled={busy || !!scrollRegion}
              onClick={() => {
                remember()
                setReady(false)
                setSelection(
                  initial.workArea ?? {
                    x: 0,
                    y: 0,
                    width: initial.width,
                    height: initial.height,
                  },
                )
                setAnnotations([])
                setSelected(null)
                setTool('select')
                setCropSelected(false)
              }}
            >
              <Crop size={16} strokeWidth={1.5} />
            </button>
            <i />
            <button
              aria-label="置顶截图"
              title="置顶截图"
              disabled={busy}
              onClick={() => void output('pin')}
            >
              <Pin size={16} strokeWidth={1.5} />
            </button>
            <button
              aria-label="翻译（即将上线）"
              title="翻译 · 暂未开放"
              disabled
            >
              <Languages size={16} strokeWidth={1.5} />
            </button>
            <button
              aria-label="滚动截图"
              title="滚动截图 · 追加下一屏"
              disabled={busy}
              onClick={() => void scroll()}
            >
              <ScanLine size={16} strokeWidth={1.5} />
            </button>
            <i />
            <button
              aria-label="保存截图"
              title="保存 PNG"
              disabled={busy}
              onClick={() => void output()}
            >
              <Download size={16} strokeWidth={1.5} />
            </button>
            <button
              aria-label="复制截图"
              title="复制截图"
              disabled={busy}
              onClick={() => void output('copy')}
            >
              <Copy size={16} strokeWidth={1.5} />
            </button>
            <button
              className="screenshot-close"
              aria-label="取消截图"
              title="取消 · Esc"
              disabled={busy}
              onClick={() => void CancelScreenshot()}
            >
              <X size={16} strokeWidth={1.5} />
            </button>
          </div>
          {optionTool && (
            <div className="screenshot-options" aria-label="当前工具选项">
              <div className="screenshot-colors" role="group" aria-label="颜色">
                {colors.map((c, index) => (
                  <button
                    key={c}
                    aria-label={`颜色 ${c}`}
                    title={
                      ['红色', '橙色', '绿色', '蓝色', '紫色', '白色', '黑色'][
                        index
                      ]
                    }
                    aria-pressed={color === c}
                    className={`screenshot-color ${color === c ? 'chosen' : ''}`}
                    style={{ '--swatch': c } as React.CSSProperties}
                    onClick={() => configure({ color: c })}
                  />
                ))}
              </div>
              <i />
              {optionTool === 'text' ? (
                <div
                  className="screenshot-size-options"
                  role="group"
                  aria-label="文字字号"
                >
                  {[
                    { label: '小', value: 18 },
                    { label: '中', value: 26 },
                    { label: '大', value: 38 },
                  ].map((item) => (
                    <button
                      key={item.value}
                      aria-label={`字号：${item.label}`}
                      aria-pressed={fontSize === item.value}
                      className={fontSize === item.value ? 'active' : ''}
                      onClick={() => configure({ fontSize: item.value })}
                    >
                      {item.label}
                    </button>
                  ))}
                </div>
              ) : (
                <div
                  className="screenshot-strokes"
                  role="group"
                  aria-label="线条粗细"
                >
                  {[
                    { label: '小', value: 2, dot: 4 },
                    { label: '中', value: 4, dot: 8 },
                    { label: '大', value: 7, dot: 12 },
                  ].map((item) => (
                    <button
                      key={item.value}
                      title={item.label}
                      aria-label={`粗细：${item.label}`}
                      aria-pressed={stroke === item.value}
                      className={stroke === item.value ? 'active' : ''}
                      onClick={() => configure({ stroke: item.value })}
                    >
                      <span style={{ width: item.dot, height: item.dot }} />
                    </button>
                  ))}
                </div>
              )}
            </div>
          )}
          {(busy || message) && (
            <div className="screenshot-feedback" role="status">
              {busy ? '处理中…' : message}
            </div>
          )}
        </div>
      )}
    </div>
  )
}
