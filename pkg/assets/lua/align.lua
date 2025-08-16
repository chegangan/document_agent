-- align.lua (docx + pdf)
-- 功能：
-- - 对带 {align=center|right} 的 Div：
--   * PDF (latex)：包在 \begin{center}/\begin{flushright}
--   * DOCX：输出 Raw OpenXML 段落，设置 <w:jc w:val="center|right">
-- - 兼容 style="text-align:..." 写法
-- - 映射常见行内样式到 OpenXML：Strong/Emph/Code/Link/Subscript/Superscript 等

local function has_align_style(style, key)
  if not style then return false end
  style = style:lower()
  return style:match("text%-align%s*:%s*" .. key) ~= nil
end

local function get_align(el)
  -- 1) 优先读 {align=right|center}
  if el.attributes and el.attributes.align then
    local v = el.attributes.align:lower()
    if v == "right" or v == "center" then
      return v
    end
  end
  -- 2) 兼容旧的 style="text-align:..."
  local style = el.attributes and el.attributes.style
  if has_align_style(style, "right") then return "right" end
  if has_align_style(style, "center") then return "center" end
  return nil
end

----------------------------------------------------------------------
-- OpenXML utils
----------------------------------------------------------------------

local function xml_escape(s)
  s = s:gsub("&", "&amp;")
  s = s:gsub("<", "&lt;")
  s = s:gsub(">", "&gt;")
  s = s:gsub('"', "&quot;")
  s = s:gsub("'", "&apos;")
  return s
end

-- 生成一个 <w:r>（可带样式）
local function make_run(t, opt)
  opt = opt or {}
  local rPr = {}

  if opt.font then
    table.insert(rPr, string.format(
      '<w:rFonts w:ascii="%s" w:hAnsi="%s" w:eastAsia="%s"/>',
      opt.font, opt.font, opt.font
    ))
  end
  if opt.bold then table.insert(rPr, '<w:b/>') end
  if opt.ital then table.insert(rPr, '<w:i/>') end
  if opt.code then
    -- 代码用等宽字体（可按需替换）
    table.insert(rPr,
      '<w:rFonts w:ascii="Consolas" w:hAnsi="Consolas" w:eastAsia="Consolas"/>')
  end
  if opt.vert == "sup" then table.insert(rPr, '<w:vertAlign w:val="superscript"/>') end
  if opt.vert == "sub" then table.insert(rPr, '<w:vertAlign w:val="subscript"/>') end

  local tnode = string.format('<w:t xml:space="preserve">%s</w:t>', xml_escape(t))
  local rPrXml = (#rPr > 0) and ('<w:rPr>' .. table.concat(rPr) .. '</w:rPr>') or ''
  return '<w:r>' .. rPrXml .. tnode .. '</w:r>'
end

-- 把 Pandoc inlines 转成 <w:r>...（支持常见类型）
local function inlines_to_openxml(inls)
  local xml = {}
  local function emit_text(s) table.insert(xml, make_run(s)) end

  local function handle_inline(inl, style)
    local t = inl.t
    if t == 'Str' then
      table.insert(xml, make_run(inl.text, style))
    elseif t == 'Space' then
      table.insert(xml, make_run(' ', style))
    elseif t == 'SoftBreak' or t == 'LineBreak' then
      -- docx 段落内换行：用 w:br
      table.insert(xml, '<w:r><w:br/></w:r>')
    elseif t == 'Strong' then
      for _, c in ipairs(inl.c) do handle_inline(c, {bold=true}) end
    elseif t == 'Emph' then
      for _, c in ipairs(inl.c) do handle_inline(c, {ital=true}) end
    elseif t == 'Code' then
      table.insert(xml, make_run(inl.text, {code=true}))
    elseif t == 'Subscript' then
      for _, c in ipairs(inl.c) do handle_inline(c, {vert='sub'}) end
    elseif t == 'Superscript' then
      for _, c in ipairs(inl.c) do handle_inline(c, {vert='sup'}) end
    elseif t == 'Quoted' then
      -- 简单处理引号
      local q = inl.c[1]
      local inner = inl.c[2]
      local lq, rq = '“','”'
      if q == 'SingleQuote' then lq, rq = '‘','’' end
      table.insert(xml, make_run(lq))
      for _, c in ipairs(inner) do handle_inline(c, style) end
      table.insert(xml, make_run(rq))
    elseif t == 'Link' then
      -- 用超链接 run
      local txt = pandoc.utils.stringify(inl.c[1])
      local url = inl.c[2][1]
      local run = make_run(txt)
      table.insert(xml,
        '<w:hyperlink r:id="rId1" w:history="1">' .. run .. '</w:hyperlink>')
      -- 注：严格来说应创建关系 rId，但 Pandoc 在大多数场景会接管链接，简化处理
    else
      -- 兜底：转成纯文本
      local s = pandoc.utils.stringify(inl)
      if s and s ~= '' then table.insert(xml, make_run(s)) end
    end
  end

  for _, inl in ipairs(inls) do handle_inline(inl, {}) end
  return table.concat(xml)
end

-- 输出一个对齐好的段落 openxml
local function para_openxml_with_alignment(inlines, align, font)
  local jc = string.format('<w:jc w:val="%s"/>', align)  -- center / right
  local rPr = '' -- 段落级别的字体等可按需加
  local runs = inlines_to_openxml(inlines)
  local xml = table.concat({
    '<w:p>',
      '<w:pPr>', jc, rPr, '</w:pPr>',
      runs,
    '</w:p>'
  })
  return pandoc.RawBlock('openxml', xml)
end

----------------------------------------------------------------------
-- 主逻辑
----------------------------------------------------------------------

function Div(el)
  local a = get_align(el)
  if not a then return nil end

  if FORMAT:match("latex") then
    if a == "center" then
      return { pandoc.RawBlock("latex", "\\begin{center}"), el, pandoc.RawBlock("latex", "\\end{center}") }
    elseif a == "right" then
      return { pandoc.RawBlock("latex", "\\begin{flushright}"), el, pandoc.RawBlock("latex", "\\end{flushright}") }
    end
    return nil
  end

  if FORMAT:match("docx") then
    -- 取 Div 里的第一层块（常见是单个 Para）；把其中的 inlines 取出，合成一个对齐段落
    -- 若 Div 里有多个块，就逐块输出（每块一个对齐段落）
    local out = {}
    for _, blk in ipairs(el.content) do
      if blk.t == 'Para' or blk.t == 'Plain' then
        table.insert(out, para_openxml_with_alignment(blk.c, a))
      elseif blk.t == 'Header' then
        -- Header 也当段落处理
        table.insert(out, para_openxml_with_alignment(blk.c, a))
      else
        -- 其他块（List/CodeBlock等）按原样放在对齐段落里并不合适，这里先原样输出
        table.insert(out, blk)
      end
    end
    return out
  end

  -- 其他格式（HTML等）保留 Div 原样（让 writer 自己处理）
  return nil
end
