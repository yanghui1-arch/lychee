import { test, expect } from 'bun:test'
import { translateRegions } from '../src/features/screenshot/translation'
import { mergeTextPixels, paintTranslation, verticalTextLayout } from '../src/features/screenshot/translation'

test('glyphs render over antialiasing without rejecting the group; other pixels remain intact', () => {
  const original=new Uint8ClampedArray([255,255,255,255,0,0,0,255,255,0,255,255,125,125,125,255])
  const blank=new Uint8ClampedArray(16)
  const result=mergeTextPixels(original,blank,4,[{x:1,y:0,width:1,color:[255,255,255]}],[255,255,255])!
  expect(Array.from(result)).toEqual([255,255,255,255,255,255,255,255,255,0,255,255,125,125,125,255])
  const glyphs=new Uint8ClampedArray(16);glyphs[11]=255
  const painted=mergeTextPixels(original,glyphs,4,[{x:1,y:0,width:1,color:[255,255,255]}],[255,255,255])
  expect(Array.from(painted.slice(8,12))).toEqual([0,0,0,255])
  expect(Array.from(painted.slice(12))).toEqual([125,125,125,255])
  expect(original[4]).toBe(0)
})

test('a fitting translation renders without a cleanup mask and uses available width', () => {
  const previous=globalThis.document
  let writes=0
  const words:string[]=[]
  const ink={font:'',textBaseline:'',fillStyle:'',save(){},restore(){},beginPath(){},rect(){},clip(){},fillText(text:string){words.push(text)},getImageData(_x:number,_y:number,w:number,h:number){return {data:new Uint8ClampedArray(w*h*4)}}}
  globalThis.document={createElement:()=>({width:0,height:0,getContext:()=>ink})} as unknown as Document
  try {
    const ctx={font:'',save(){},restore(){},measureText(text:string){return {width:text.length*parseFloat(this.font.split(' ')[1])/2}},getImageData(_x:number,_y:number,w:number,h:number){return {data:new Uint8ClampedArray(w*h*4).fill(255)}},putImageData(){writes++}} as unknown as CanvasRenderingContext2D
    const result=paintTranslation(ctx,{x:.25,y:.25,width:30,height:20,text:'test',score:1},'这是一段完整的译文',{bounds:{x:0,y:0,width:120,height:24},obstacles:[]})
    expect(result.fits).toBe(true)
    expect(result.reason).toBe('success')
    expect(writes).toBe(1)
    expect(words.join('')).toBe('这是一段完整的译文')
  } finally {globalThis.document=previous}
})

test('right-edge Ad label borrows only the left blank space it needs', () => {
  const previous=globalThis.document
  const positions:number[]=[]
  const ink={font:'',textBaseline:'',fillStyle:'',save(){},restore(){},beginPath(){},rect(){},clip(){},fillText(_text:string,x:number){positions.push(x)},getImageData(_x:number,_y:number,w:number,h:number){return {data:new Uint8ClampedArray(w*h*4)}}}
  globalThis.document={createElement:()=>({getContext:()=>ink})} as unknown as Document
  try {
    const ctx={font:'',save(){},restore(){},measureText(text:string){return {width:text==='汉Ag'?100:text==='广告'?30:18}},getImageData(_x:number,_y:number,w:number,h:number){return {data:new Uint8ClampedArray(w*h*4).fill(255)}},putImageData(){}} as unknown as CanvasRenderingContext2D
    const result=paintTranslation(ctx,{x:80,y:0,width:18,height:18,text:'Ad',score:1},'广告',{bounds:{x:0,y:0,width:100,height:18},obstacles:[]})
    expect(result.fits).toBe(true)
    expect(result.lines).toBe(1)
    expect(result.availableWidth).toBe(30)
    const blocked=paintTranslation(ctx,{x:80,y:0,width:18,height:18,text:'Ad',score:1},'广告',{bounds:{x:0,y:0,width:100,height:18},obstacles:[{x:0,y:0,width:80,height:18}]})
    expect(blocked.fits).toBe(false)
  } finally {globalThis.document=previous}
})

test('old coloured glyph edges are removed before blending new antialiasing', () => {
  const original=new Uint8ClampedArray([80,120,200,255,255,150,20,255])
  const glyphs=new Uint8ClampedArray([0,0,0,128,0,0,0,0])
  const output=mergeTextPixels(original,glyphs,2,[{x:0,y:0,width:2,color:[20,20,20]}])
  expect(Array.from(output)).toEqual([10,10,10,255,20,20,20,255])
})

test('LOL can expand past its own source ink without covering a left neighbour', () => {
  const previous=globalThis.document
  const ink={font:'',textBaseline:'',fillStyle:'',save(){},restore(){},beginPath(){},rect(){},clip(){},fillText(){},getImageData(_x:number,_y:number,w:number,h:number){return {data:new Uint8ClampedArray(w*h*4)}}}
  globalThis.document={createElement:()=>({getContext:()=>ink})} as unknown as Document
  try {
    const ctx={font:'',save(){},restore(){},measureText(text:string){return {width:text.length*parseFloat(this.font.split(' ')[1])}},getImageData(x:number,_y:number,w:number,h:number){const data=new Uint8ClampedArray(w*h*4).fill(24);for(let row=0;row<h;row++)for(let col=0;col<w;col++){const i=(row*w+col)*4;data[i+3]=255;if(x+col>=70&&x+col<72){data[i]=240;data[i+1]=180;data[i+2]=40}}return {data}},putImageData(){}} as unknown as CanvasRenderingContext2D
    const area={bounds:{x:0,y:0,width:120,height:20},obstacles:[{x:0,y:0,width:50,height:20}]}
    const region={x:50,y:0,width:20,height:20,text:'LOL',score:1}
    expect(paintTranslation(ctx,region,'哈哈',area).fits).toBe(false)
    const erase=Array.from({length:20},(_,y)=>({x:20,y,width:2,color:[24,24,24]}))
    const result=paintTranslation(ctx,{...region,erase},'哈哈',area)
    expect(result.fits).toBe(true)
    expect(result.lines).toBe(1)
    expect(result.availableWidth).toBeGreaterThan(20)
  } finally {globalThis.document=previous}
})

test('11/12-character short translations fit the logged 166/186px widths at unchanged font height', () => {
  const previous=globalThis.document
  const limits:number[]=[]
  const ink={font:'',textBaseline:'',fillStyle:'',save(){},restore(){},beginPath(){},rect(){},clip(){},fillText(_text:string,_x:number,_y:number,maxWidth:number){limits.push(maxWidth)},getImageData(_x:number,_y:number,w:number,h:number){return {data:new Uint8ClampedArray(w*h*4)}}}
  globalThis.document={createElement:()=>({getContext:()=>ink})} as unknown as Document
  try {
    const ctx={font:'',save(){},restore(){},measureText(text:string){const size=parseFloat(this.font.split(' ')[1]);return {width:text.length*size,actualBoundingBoxAscent:size*.85,actualBoundingBoxDescent:0}},getImageData(_x:number,_y:number,w:number,h:number){return {data:new Uint8ClampedArray(w*h*4).fill(24)}},putImageData(){}} as unknown as CanvasRenderingContext2D
    for(const [width,count] of [[166,11],[186,12]]) {
      const result=paintTranslation(ctx,{x:0,y:0,width,height:17,text:'original',score:1},'字'.repeat(count),{bounds:{x:0,y:0,width,height:18},obstacles:[]})
      expect(result.fits).toBe(true)
      expect(result.fontSize).toBe(17)
      expect(result.lines).toBe(1)
      expect(limits.at(-1)).toBe(width)
    }
  } finally {globalThis.document=previous}
})

test('glyph blocks are vertically centered with space for antialiasing, including multiline and tight slots', () => {
  expect(verticalTextLayout(20,14,30)).toEqual({height:20,offset:3})
  expect(verticalTextLayout(18,34,50)).toEqual({height:36,offset:1})
  expect(verticalTextLayout(18,17,18)).toEqual({height:18,offset:.5})
  const original=globalThis.document
  const baselines:number[]=[]
  const ink={font:'',textBaseline:'',fillStyle:'',save(){},restore(){},beginPath(){},rect(){},clip(){},fillText(_text:string,_x:number,y:number){baselines.push(y)},getImageData(_x:number,_y:number,w:number,h:number){return {data:new Uint8ClampedArray(w*h*4)}}}
  globalThis.document={createElement:()=>({getContext:()=>ink})} as unknown as Document
  try {
    const ctx={font:'',save(){},restore(){},measureText(text:string){return {width:text.length*8,actualBoundingBoxAscent:this.font.includes('100px')?80:12,actualBoundingBoxDescent:this.font.includes('100px')?20:2}},getImageData(_x:number,_y:number,w:number,h:number){return {data:new Uint8ClampedArray(w*h*4).fill(24)}},putImageData(){}} as unknown as CanvasRenderingContext2D
    const result=paintTranslation(ctx,{x:0,y:0,width:100,height:20,text:'source',score:1},'居中')
    expect(result.fits).toBe(true)
    expect(baselines).toEqual([15]) // 3px top margin + 12px actual ascent.
  } finally {globalThis.document=original}
})

test('bounded workers apply completion order and continue after a failed region', async () => {
  const regions = Array.from({length: 9}, (_, i) => ({text: String(i), score: 1, x: 0, y: 0, width: 20, height: 10}))
  let active = 0, peak = 0, failures = 0
  const completed: string[] = []
  await translateRegions(regions, async region => {
    active++; peak = Math.max(peak, active)
    await new Promise(resolve => setTimeout(resolve, region.text === '0' ? 60 : 2))
    active--
    if (region.text === '2') throw new Error('server unavailable')
    return region.text
  }, (_, text) => completed.push(text), () => failures++)
  expect(peak).toBe(4)
  expect(completed[0]).toBe('1')
  expect(completed.at(-1)).toBe('0')
  expect(completed.length).toBe(8)
  expect(failures).toBe(1)
})
