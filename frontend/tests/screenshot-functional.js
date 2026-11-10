export async function runScreenshotFunctionalTests() {
  const out = [],
    wait = () => new Promise((r) => setTimeout(r, 45)),
    assert = (ok, name) => {
      if (!ok) throw Error(name)
      out.push(name)
    }
  const button = (name) =>
      document.querySelector(`button[aria-label="${name}"]`),
    click = async (name) => {
      const b = button(name)
      if (!b) throw Error('Missing ' + name)
      b.click()
      await wait()
    }
  const key = async (k) => {
    window.dispatchEvent(
      new KeyboardEvent('keydown', { key: k, ctrlKey: true, bubbles: true }),
    )
    await wait()
  }
  const handles = () => document.querySelectorAll('.screenshot-handle').length
  const size = () => document.querySelector('.screenshot-size')?.textContent
  const drag = async (x, y, x2, y2, target, during) => {
    const el = document.querySelector('.screenshot-stage'),
      b = el.getBoundingClientRect(),
      s = b.width / 1280
    const send = (t, a, c, node = el) =>
      node.dispatchEvent(
        new PointerEvent(t, {
          bubbles: true,
          pointerId: 1,
          button: 0,
          clientX: b.left + a * s,
          clientY: b.top + c * s,
        }),
      )
    send('pointerdown', x, y, target || el)
    await wait()
    send('pointermove', x2, y2)
    await wait()
    if (during) during()
    send('pointerup', x2, y2)
    await wait()
  }
  const rows = () =>
    new Set(
      [...document.querySelectorAll('.screenshot-tools button')].map((b) =>
        Math.round(b.getBoundingClientRect().y),
      ),
    ).size
  assert(size() === '1280 × 752', 'default preview excludes taskbar')
  assert(
    !document.querySelector('.screenshot-dock') && !handles(),
    'preview has no toolbar or handles',
  )
  const css = getComputedStyle(document.querySelector('.screenshot-selection'))
  assert(
    css.boxSizing === 'border-box' && parseFloat(css.borderLeftWidth) >= 3,
    'thicker border is inset and visible at full-screen edges',
  )
  await drag(100, 100, 800, 600, null, () =>
    assert(!handles(), 'no crop handles during drawing'),
  )
  assert(
    !!document.querySelector('.screenshot-dock') && !handles(),
    'release shows toolbar without selecting crop handles',
  )
  assert(
    !button('移动 / 调整') && !document.querySelector('.screenshot-options'),
    'no move icon or global settings',
  )
  assert(
    rows() === 1 &&
      button('取消截图').parentElement.classList.contains('screenshot-tools'),
    'exit is in single main row',
  )
  await drag(120, 120, 120, 120)
  assert(handles() === 8, 'explicitly clicking crop reveals handles')
  await drag(800, 600, 900, 650, document.querySelector('[data-corner="se"]'))
  assert(size() === '800 × 550', 'resize selected crop')
  await key('z')
  assert(size() === '700 × 500', 'undo crop resize')
  await key('y')
  assert(size() === '800 × 550', 'redo crop resize')
  const before = document
    .querySelector('.screenshot-dock')
    .getBoundingClientRect()
  await drag(120, 120, 170, 150)
  const after = document
    .querySelector('.screenshot-dock')
    .getBoundingClientRect()
  assert(before.x !== after.x || before.y !== after.y, 'toolbar follows crop')
  assert(
    after.left >= 0 &&
      after.top >= 0 &&
      after.right <= innerWidth &&
      after.bottom <= innerHeight,
    'toolbar remains within viewport',
  )
  await key('z')
  await click('矩形')
  assert(
    !!document.querySelector('.screenshot-options') &&
      !document.querySelector('select'),
    'shape options use direct controls',
  )
  assert(
    document.querySelector('.screenshot-options').getBoundingClientRect().top >=
      document.querySelector('.screenshot-tools').getBoundingClientRect()
        .bottom,
    'options are below main row',
  )
  assert(
    button('粗细：小') &&
      button('粗细：中') &&
      button('粗细：大') &&
      !button('字号：小'),
    'shape shows only relevant thickness controls',
  )
  const dots = ['小', '中', '大'].map(
    (n) =>
      button('粗细：' + n)
        .querySelector('span')
        .getBoundingClientRect().width,
  )
  assert(
    dots[0] < dots[1] &&
      dots[1] < dots[2] &&
      !document.querySelector('.screenshot-options').textContent.includes('px'),
    'three solid dot sizes replace pixel values',
  )
  await click('粗细：大')
  await click('颜色 #3b82f6')
  await drag(160, 160, 360, 280, null, () =>
    assert(!handles(), 'rectangle has no points while drawing'),
  )
  assert(!handles(), 'rectangle has no points after drawing')
  await click('矩形')
  assert(
    !document.querySelector('.screenshot-options'),
    'clicking active tool cancels it',
  )
  await drag(
    200,
    200,
    200,
    200,
    document.querySelector('rect[data-annotation]'),
  )
  assert(handles() === 8, 'selecting rectangle reveals points')
  await drag(360, 280, 420, 320, document.querySelector('[data-corner="se"]'))
  assert(
    +document.querySelector('rect[data-annotation]').getAttribute('width') ===
      260,
    'resize explicitly selected rectangle',
  )
  await click('圆形')
  assert(
    button('粗细：中').getAttribute('aria-pressed') === 'true' &&
      button('颜色 #f43f5e').getAttribute('aria-pressed') === 'true',
    'tools retain independent settings',
  )
  await drag(450, 180, 580, 310, null, () =>
    assert(!handles(), 'ellipse has no points while drawing'),
  )
  assert(!handles(), 'ellipse has no points after drawing')
  await click('圆形')
  await drag(
    500,
    220,
    500,
    220,
    document.querySelector('ellipse[data-annotation]'),
  )
  assert(handles() === 8, 'selecting ellipse reveals points')
  await click('箭头')
  await drag(650, 450, 400, 350)
  assert(
    !handles() &&
      +document.querySelector('line[data-annotation]').getAttribute('x2') ===
        400,
    'arrow follows drag with no points',
  )
  await key('z')
  assert(!document.querySelector('line[data-annotation]'), 'undo drawing')
  await key('y')
  assert(!!document.querySelector('line[data-annotation]'), 'redo drawing')
  await click('文字')
  assert(
    button('字号：小') &&
      button('字号：中') &&
      button('字号：大') &&
      !button('粗细：小'),
    'text shows direct font sizes without thickness',
  )
  await click('字号：大')
  await click('颜色 #a855f7')
  await drag(300, 480, 300, 480)
  const input = document.querySelector('textarea')
  assert(
    input &&
      document.activeElement === input &&
      getComputedStyle(input).borderTopStyle === 'dashed',
    'single click focuses dashed text box',
  )
  const setValue = (value) => {
    Object.getOwnPropertyDescriptor(
      HTMLTextAreaElement.prototype,
      'value',
    ).set.call(input, value)
    input.dispatchEvent(new Event('input', { bubbles: true }))
  }
  setValue('第一行')
  await wait()
  const enter = new KeyboardEvent('keydown', {
    key: 'Enter',
    bubbles: true,
    cancelable: true,
  })
  input.dispatchEvent(enter)
  await wait()
  assert(
    !enter.defaultPrevented && !!document.querySelector('textarea'),
    'Enter is newline instead of confirmation',
  )
  setValue('第一行\n第二行\n第三行')
  await wait()
  assert(
    getComputedStyle(input).overflowY === 'hidden' &&
      input.clientHeight >= input.scrollHeight,
    'multiline text auto-grows without scrollbars',
  )
  input.dispatchEvent(new FocusEvent('focusout', { bubbles: true }))
  await wait()
  assert(
    !document.querySelector('textarea') &&
      document.querySelector('text[data-annotation]').textContent ===
        '第一行第二行第三行',
    'blur finishes text and removes dashed box',
  )
  assert(!handles(), 'text does not automatically show points')
  await key('z')
  assert(!document.querySelector('text[data-annotation]'), 'undo text')
  await key('y')
  assert(!!document.querySelector('text[data-annotation]'), 'redo text')
  await click('文字')
  assert(
    !document.querySelector('.screenshot-options') && rows() === 1,
    'cancel tool returns to one clean row',
  )
  await click('矩形')
  assert(
    button('粗细：大').getAttribute('aria-pressed') === 'true' &&
      button('颜色 #3b82f6').getAttribute('aria-pressed') === 'true',
    'returning to rectangle restores its settings',
  )
  await click('保存截图')
  const image = new Image()
  image.src = window.exported
  await image.decode()
  assert(
    image.width === 800 && image.height === 550,
    'PNG exports selection and multiline annotations',
  )
  assert(button('翻译（即将上线）').disabled, 'translation remains placeholder')
  await click('置顶截图')
  assert(!!document.querySelector('.screenshot-pin'), 'pin screenshot')
  await click('关闭置顶截图')
  assert(window.calls.includes('cancel'), 'close pinned screenshot')
  return out
}
