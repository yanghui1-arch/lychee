export async function runPinPositionTest() {
  const wait=()=>new Promise(resolve=>setTimeout(resolve,70))
  await wait()
  const stage=document.querySelector('.screenshot-stage')
  const bounds=stage.getBoundingClientRect(),scale=bounds.width/1280
  const send=(type,x,y)=>stage.dispatchEvent(new PointerEvent(type,{bubbles:true,pointerId:1,button:0,clientX:bounds.left+x*scale,clientY:bounds.top+y*scale}))
  send('pointerdown',400,200);await wait();send('pointermove',700,380);await wait();send('pointerup',700,380);await wait()
  document.querySelector('button[aria-label="置顶截图"]').click()
  for(let i=0;i<30&&!document.querySelector('.screenshot-pin');i++)await wait()
  const call=window.calls.find(call=>Array.isArray(call)&&call[0]==='pin')
  if(JSON.stringify(call)!==JSON.stringify(['pin',400,200,300,180]))throw Error('wrong desktop anchor: '+JSON.stringify(call))
  if(!document.querySelector('.screenshot-pin'))throw Error('pinned image not shown')
  return {anchor:[400,200],size:[300,180],pinned:true}
}
