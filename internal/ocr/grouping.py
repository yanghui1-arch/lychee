"""Group OCR lines within detected text blocks, preserving every source rectangle."""
import re
import unicodedata

MESSAGE = re.compile(r"^\s*(?:\d{1,2}:\d{2}|[^\s:：]{1,60}\s*[:：])")
TEXT_LABELS = {"text", "paragraph_title", "doc_title", "document_title", "title", "list", "sidebar_text"}

DATE = re.compile(r"^(?:\d{1,4}年)?\d{1,2}月\d{1,2}日$|^(?:Jan|Feb|Mar|Apr|May|Jun|Jul|Aug|Sep|Oct|Nov|Dec)[a-z]*\s+\d{1,2}(?:,?\s+\d{4})?$",re.I)

def chat_prefix(text, box=None, layout=()):
    # Headings may contain colons. Only a compact chat handle is a prefix.
    match=re.match(r"^\s*(?:\d{1,2}:\d{2}\s*)?[^\s:：]{1,40}[:：]\s*",text)
    if not match: return None
    if box is not None:
        x,y,right,bottom=box
        for block in layout:
            if block['label'] not in {'doc_title','document_title','paragraph_title','title'}: continue
            a,b,c,d=block['coordinate']
            overlap=max(0,min(right,c)-max(x,a))*max(0,min(bottom,d)-max(y,b))
            if overlap >= (right-x)*(bottom-y)*.5: return None
    return match

def author_above_date(text, box, dates):
    if not re.fullmatch(r"[^\W\d_]+(?:[ .’'\-]+[^\W\d_]+){0,3}",text.strip()): return False
    x,y,right,bottom=box
    for dx,dy,dr,db in dates:
        height=max(bottom-y,db-dy)
        if bottom-y <= (db-dy)*1.5 and 0 <= dy-bottom <= height*.8 and abs(x-dx)<=height*1.5:
            return True
    return False

def recognized_word(word):
    return bool(word.strip()) and (any(c.isalnum() for c in word) or all(unicodedata.category(c).startswith("P") or c in "*`~" for c in word if not c.isspace()))

def group_regions(regions, layout):
    assignments = []
    for r in regions:
        candidates = []
        for i, block in enumerate(layout):
            if block["label"] not in TEXT_LABELS:
                continue
            x1, y1, x2, y2 = block["coordinate"]
            overlap = max(0, min(r["x"]+r["width"], x2)-max(r["x"], x1)) * max(0, min(r["y"]+r["height"], y2)-max(r["y"], y1))
            if overlap >= r["width"]*r["height"]*.6:
                candidates.append(((x2-x1)*(y2-y1), i))
        assignments.append(min(candidates)[1] if candidates else None)
    rows = []
    for index in sorted(range(len(regions)), key=lambda i: regions[i]["y"]):
        r = regions[index]
        if rows:
            first = regions[rows[-1][0]]
            same_row = abs(r["y"]+r["height"]/2-first["y"]-first["height"]/2) <= .45*max(r["height"],first["height"])
        else:
            same_row = False
        if same_row: rows[-1].append(index)
        else: rows.append([index])
    order = [index for row in rows for index in sorted(row,key=lambda i: regions[i]["x"])]
    groups = []
    for index in order:
        r = regions[index]
        target = None
        # OCR can put a closing quote/asterisk on its own tiny line. It belongs
        # to the previous message's cleanup, not a separate translation request.
        if re.fullmatch(r"[\s*`\"“”‘’]+",r["text"]):
            for group in reversed(groups):
                parent=regions[group["members"][-1]]
                left=min(regions[i].get("flow_left",regions[i]["x"]) for i in group["members"])
                right=max(regions[i]["x"]+regions[i]["width"] for i in group["members"])
                gap=r["y"]-parent["y"]-parent["height"]
                if -.3*parent["height"]<=gap<=1.2*parent["height"] and left-parent["height"]<=r["x"]<=right+parent["height"]:
                    r["cleanup_only"]=True
                    group["members"].append(index)
                    break
            continue

        if not r.get("message_start") and not MESSAGE.match(r["text"]):
            for group in reversed(groups):
                # Layout is supporting evidence; nearby continuation lines can cross a missed block.
                if group["block"] is not None and assignments[index] is not None and group["block"] != assignments[index] and not any(regions[i].get("message_start") or MESSAGE.match(regions[i]["text"]) for i in group["members"]):
                    continue
                prev = regions[group["members"][-1]]
                h = max(prev["height"], r["height"])
                chat = any(regions[i].get("message_start") or MESSAGE.match(regions[i]["text"]) for i in group["members"])
                similar_style = min(prev["height"],r["height"]) >= (.6 if chat else .8)*h
                separated_ui = (r["text"].lstrip().startswith('#') or prev["text"].lstrip().startswith('#') or
                                re.fullmatch(r"[\d\s年月日/.-]+",r["text"]) or re.fullmatch(r"[\d\s年月日/.-]+",prev["text"]))
                if not similar_style or separated_ui:
                    break
                same_line = abs((prev["y"]+prev["height"]/2)-(r["y"]+r["height"]/2)) <= .45*h
                inline = same_line and 0 <= r["x"]-prev["x"]-prev["width"] <= (3 if chat else .75)*h
                gap = r["y"]-prev["y"]-prev["height"]
                group_left = min(regions[i].get("flow_left",regions[i]["x"]) for i in group["members"])
                group_right = max(regions[i]["x"]+regions[i]["width"] for i in group["members"])
                vertical = (not same_line and -.2*h <= gap <= .8*h and
                            similar_style and
                            max(group_left,r["x"]) <= min(group_right,r["x"]+r["width"])+h)
                if inline or vertical:
                    target = group
                break
        if target is None:
            groups.append(dict(block=assignments[index], members=[index]))
        else:
            target["members"].append(index)
    output = []
    for group in groups:
        all_parts = [regions[i] for i in group["members"]]
        parts = [r for r in all_parts if not r.get("cleanup_only")]
        x = min(r["x"] for r in parts); y = min(r["y"] for r in parts)
        right = max(r["x"]+r["width"] for r in parts); bottom = max(r["y"]+r["height"] for r in parts)
        output.append(dict(text=" ".join(r["text"].strip() for r in parts), score=min(r["score"] for r in parts),
                           x=x,y=y,width=right-x,height=bottom-y,
                           erase=[dict(x=p["x"]-x,y=p["y"]-y,width=p["width"],color=p["color"]) for r in all_parts for p in r.get("erase",[])],
                           boxes=[dict(x=r["x"]-x,y=r["y"]-y,width=r["width"],height=r["height"]) for r in parts]))
    return output, [dict(block=g["block"],members=g["members"]) for g in groups]
