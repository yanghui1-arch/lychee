"""Local PP-OCRv6 Small worker. Image paths arrive over stdin; JSON on stdout."""
import contextlib
import json
import sys
import os
from time import perf_counter
from pathlib import Path
from grouping import group_regions, recognized_word, DATE, chat_prefix, author_above_date

# Models ship inside the desktop executable; no online source probes or fallback.
os.environ["PADDLE_PDX_DISABLE_MODEL_SOURCE_CHECK"] = "True"
os.environ["HF_HUB_OFFLINE"] = "1"
model_dir = Path(sys.argv[1])
for name in ("PP-OCRv6_small_det", "PP-OCRv6_small_rec", "PP-DocLayout-S"):
    for file in ("inference.json", "inference.pdiparams", "inference.yml"):
        if not (model_dir / name / file).is_file():
            raise FileNotFoundError(f"Bundled OCR model missing: {name}/{file}")

started = perf_counter()
with contextlib.redirect_stdout(sys.stderr):
    import cv2
    from paddleocr import PaddleOCR, LayoutDetection
    import psutil
    process = psutil.Process()
    imported = perf_counter()
    engine = PaddleOCR(
        text_detection_model_name="PP-OCRv6_small_det",
        text_recognition_model_name="PP-OCRv6_small_rec",
        text_detection_model_dir=str(model_dir / "PP-OCRv6_small_det"),
        text_recognition_model_dir=str(model_dir / "PP-OCRv6_small_rec"),
        use_doc_orientation_classify=False,
        use_doc_unwarping=False,
        use_textline_orientation=False,
        device="cpu",
        # Paddle 3.3 on Windows fails on PP-OCRv6 PIR attributes with oneDNN.
        enable_mkldnn=False,
    )
    loaded = perf_counter()
    layout_memory_before = process.memory_info().rss
    layout_engine = None
    layout_load_error = None
    try:
        layout_engine = LayoutDetection(model_name="PP-DocLayout-S", model_dir=str(model_dir / "PP-DocLayout-S"), device="cpu", enable_mkldnn=False)
    except Exception as error:
        layout_load_error = type(error).__name__
    layout_loaded = perf_counter()
    preload_metrics = dict(layout_rss_before_mb=layout_memory_before/1048576,
                           layout_rss_after_mb=process.memory_info().rss/1048576,
                           layout_rss_delta_mb=(process.memory_info().rss-layout_memory_before)/1048576)


print(json.dumps(dict(ready=True, timings_ms=dict(
    imports_ms=(imported-started)*1000,
    model_load_ms=(loaded-imported)*1000,
    layout_load_ms=(layout_loaded-loaded)*1000,
), metrics=preload_metrics, layout_error=layout_load_error)), flush=True)

for request_line in sys.stdin:
    try:
        request = json.loads(request_line)
        with contextlib.redirect_stdout(sys.stderr):
            image = cv2.imread(request["image"])
            if image is None:
                raise ValueError("Cannot decode screenshot")
            layout_started = perf_counter()
            layout = []
            layout_error = layout_load_error
            layout_memory_before = process.memory_info().rss
            if layout_engine is not None:
                try:
                    for prediction in layout_engine.predict(image):
                        layout.extend(prediction.json["res"]["boxes"])
                except Exception as error:
                    layout_error = type(error).__name__
                    layout = []
            layout_ms = (perf_counter()-layout_started)*1000
            layout_memory_after = process.memory_info().rss
            regions = []
            preserved_metadata = 0
            inference_started = perf_counter()
            for result in engine.predict(image, return_word_box=True):
                data = result.json["res"]
                dates=[b for t,b in zip(data["rec_texts"],data["rec_boxes"]) if DATE.fullmatch(t.strip())]
                for line_index, (text, score, box) in enumerate(zip(data["rec_texts"], data["rec_scores"], data["rec_boxes"])):
                    if not text.strip() or score < 0.5:
                        continue
                    import re
                    from collections import Counter
                    if len(text.strip()) == 1 and score < .9:
                        continue
                    # Date text can be occluded or missed by OCR. A short name directly
                    # above a detected heading with an adjacent avatar is still metadata.
                    profile_name=False
                    if re.fullmatch(r"[^\W\d_]+(?:[ .’'\-]+[^\W\d_]+){0,3}",text.strip()):
                        import numpy as np
                        bx,by,br,bb=map(int,box)
                        for block in layout:
                            if block['label'] not in {'doc_title','document_title','title'}: continue
                            tx,ty,tr,tb=block['coordinate']
                            if 0<=ty-bb<=max(48,(bb-by)*3) and abs(bx-tx)<=24 and bb-by<=(tb-ty)*.8:
                                avatar=image[max(0,by-4):min(image.shape[0],bb+8),max(0,bx-48):max(0,bx-6)]
                                if avatar.size and float(np.std(avatar))>25 and float(np.mean(np.min(avatar,axis=2)<180))>.15:
                                    profile_name=True
                                    break
                    # Preserve account identifiers and date metadata; they are not prose.
                    if re.fullmatch(r"[A-Za-z0-9_]*[a-z][A-Z][A-Za-z0-9_]*",text.strip()) or DATE.fullmatch(text.strip()) or author_above_date(text,box,dates) or profile_name:
                        preserved_metadata += 1
                        continue
                    prefix = chat_prefix(text,box,layout)
                    decoration = re.match(r"^[QO○◯]\s*(?=\d+\s+(?:comments|replies))",text,re.I)
                    cutoff = prefix.end() if prefix else decoration.end() if decoration else 0
                    words = data.get("text_word",[])
                    word_boxes = data.get("text_word_boxes",[])
                    if line_index >= len(words):
                        continue  # No word alignment: preserve original rather than clear a line rectangle.
                    selected = []
                    cursor = 0
                    prefix_right = 0
                    for word, word_box in zip(words[line_index],word_boxes[line_index]):
                        offset = text.find(word,cursor)
                        if offset < 0: continue
                        cursor = offset+len(word)
                        if offset < cutoff:
                            prefix_right=max(prefix_right,int(word_box[2]))
                            continue
                        if not recognized_word(word): continue
                        selected.append((word, list(map(int,word_box))))
                    if not selected: continue
                    x=min(b[0] for _,b in selected); y=min(b[1] for _,b in selected)
                    right=max(b[2] for _,b in selected); bottom=max(b[3] for _,b in selected)
                    crop=image[max(0,y):bottom,max(0,x):right]
                    if not crop.size: continue
                    colors=Counter(tuple(int(c)//8*8 for c in pixel) for pixel in crop.reshape(-1,3))
                    bg=colors.most_common(1)[0][0]
                    erase=[]
                    # Word alignment defines the cleanup scope. Include coloured ClearType
                    # edge pixels and a 1px halo; grayscale-only masks leave old glyphs visible.
                    import numpy as np
                    # CTC word boxes are approximate. Clean the complete recognized
                    # body span including gaps, otherwise word-edge strokes remain.
                    for _,(wx,wy,wr,wb) in [("",(x,y,right,bottom))]:
                        sx=max(0,prefix_right,wx-8); sy=max(0,wy-3)
                        ex=min(image.shape[1],wr+6); ey=min(image.shape[0],wb+3)
                        patch=image[sy:ey,sx:ex]
                        if not patch.size: continue
                        # Estimate the actual local background, not a quantized RGB bucket.
                        pixels=patch.reshape(-1,3)
                        buckets=pixels//8
                        keys,counts=np.unique(buckets,axis=0,return_counts=True)
                        common=keys[counts.argmax()]
                        cluster=pixels[np.all(buckets==common,axis=1)]
                        local_bg=np.median(cluster,axis=0).astype(np.uint8)
                        foreground=np.max(np.abs(patch.astype(np.int16)-local_bg.astype(np.int16)),axis=2)>4
                        support=np.zeros(foreground.shape,dtype=bool)
                        for _,(ax,ay,ar,ab) in selected:
                            support[max(0,ay-3-sy):min(ey-sy,ab+3-sy),max(0,ax-8-sx):min(ex-sx,ar+6-sx)]=True
                        # Recover complete glyph components rather than clipping strokes
                        # at approximate CTC word boundaries. Large unrelated graphics stay out.
                        count, labels, stats, _ = cv2.connectedComponentsWithStats(foreground.astype(np.uint8),8)
                        complete=np.zeros_like(foreground)
                        for component in range(1,count):
                            if stats[component,cv2.CC_STAT_HEIGHT] > (bottom-y)*1.5: continue
                            member=labels==component
                            if np.any(member & support): complete |= member
                        foreground=complete

                        for row in range(ey-sy):
                            run=None
                            for col in range(ex-sx+1):
                                eligible=col<ex-sx and bool(foreground[row,col])
                                if eligible and run is None: run=col
                                if not eligible and run is not None:
                                    erase.append(dict(x=sx+run,y=sy+row,width=col-run,color=local_bg[::-1].tolist()))
                                    run=None
                    regions.append(dict(text=text[cutoff:].strip(), score=float(score), x=x,y=y,width=right-x,height=bottom-y,
                                        message_start=bool(prefix),flow_left=int(box[0]),erase=erase))

        inference_ms = (perf_counter()-inference_started)*1000
        grouping_started = perf_counter()
        grouped, mapping = group_regions(regions, layout)
        grouping_ms = (perf_counter()-grouping_started)*1000
        print(json.dumps(dict(regions=grouped, raw_regions=regions, layout=layout, groups=mapping, layout_error=layout_error, metrics=dict(
            recognized_text_pixels=sum(p["width"] for r in regions for p in r.get("erase",[])),
            word_alignment_enabled=1,
            preserved_metadata_count=preserved_metadata,
            grouping_policy_version=4,
            layout_rss_before_mb=layout_memory_before/1048576,
            layout_rss_after_mb=layout_memory_after/1048576,
            rss_after_ocr_mb=process.memory_info().rss/1048576,
            process_peak_rss_mb=getattr(process.memory_info(), "peak_wset", 0)/1048576,
        ), timings_ms=dict(
            inference_ms=inference_ms,
            layout_ms=layout_ms,
            grouping_ms=grouping_ms,
        )), ensure_ascii=True), flush=True)
    except Exception as error:
        print(json.dumps(dict(error=type(error).__name__)), flush=True)
