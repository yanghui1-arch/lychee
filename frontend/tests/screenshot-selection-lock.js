export async function runSelectionLockTest(mode = 'success') {
  const wait = () => new Promise(resolve => setTimeout(resolve, 70))
  const assert = (value, message) => {if (!value) throw Error(message)}
  const button = name => document.querySelector(`button[aria-label="${name}"]`)
  const stage = document.querySelector('.screenshot-stage')
  const rect = () => document.querySelector('.screenshot-selection').getAttribute('style')
  const drag = async (x,y,x2,y2) => {
    const bounds=stage.getBoundingClientRect(),scale=bounds.width/1280
    const send=(type,a,b)=>stage.dispatchEvent(new PointerEvent(type,{bubbles:true,pointerId:1,button:0,clientX:bounds.left+a*scale,clientY:bounds.top+b*scale}))
    send('pointerdown',x,y); await wait(); send('pointermove',x2,y2);await wait();send('pointerup',x2,y2);await wait()
  }
  await wait()
  await drag(100,100,800,600)
  const before = rect()
  await drag(400,300,430,330)
  assert(rect()!==before,'crop must remain movable before translation')
  const locked = rect()
  window.ocrPending=true
  window.ocrFailure=mode==='failure'
  window.ocrRegions=[{x:0,y:0,width:100,height:24,text:'hello',score:1}]
  const originalFetch=window.fetch
  window.fetch=async (url,...args)=>url==='/api/translate'?new Response(JSON.stringify({translated_text:'你好'}),{headers:{'Content-Type':'application/json'}}):originalFetch(url,...args)
  try {
    button('翻译截图').click();await wait()
    assert(button('重新选择区域').disabled,'reselect disabled immediately')
    assert(!document.querySelector('.screenshot-handle'),'crop resize handles removed')
    await drag(400,300,500,350)
    assert(rect()===locked,'crop locked while OCR pending')
    window.finishOCR()
    for(let i=0;i<40&&button('翻译截图').disabled;i++)await wait()
    assert(!button('翻译截图').disabled,'translation should settle')
    if(mode==='failure') assert(document.querySelector('.screenshot-feedback')?.textContent.includes('OCR fixture failure'),'fixture must exercise failed OCR')
    await drag(400,300,600,500)
    await drag(0,0,80,80)
    assert(rect()===locked,'inside and outside drags must not change crop')
    for(const key of ['z','y']) {window.dispatchEvent(new KeyboardEvent('keydown',{key,ctrlKey:true,bubbles:true}));await wait();assert(rect()===locked,'history cannot restore old crop')}
    assert(button('重新选择区域').disabled&&button('滚动截图').disabled,'geometry-changing tools remain disabled')
    if(mode==='success') {button('翻译截图').click();await wait();await drag(400,300,500,400);assert(rect()===locked,'restoring original keeps crop locked')}

    return {mode,locked:true,checks:['before','pending','completed/failed','inside/outside drag','handles','history','reselect','scroll','restore']}
  } finally {window.fetch=originalFetch}
}
