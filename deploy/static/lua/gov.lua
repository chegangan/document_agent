-- gov.lua : 将抬头三段转换为 DOCX OpenXML 段落
-- 识别以下 fenced div（注意是 class，不是 custom-style）：
--   ::: {.GovTitle}    标题（红色 36pt 加粗，居中）
--   ::: {.GovDocNo}    文号（黑色 16pt，居中；可调）
--   ::: {.GovRedLine}  上边框红线（2pt）

local function stringify_inlines(inlines)
  return pandoc.utils.stringify(inlines)
end

-- 兼容老版本 pandoc：自己实现 XML 转义
local function xml_escape(s)
  s = s:gsub("&", "&amp;")
  s = s:gsub("<", "&lt;")
  s = s:gsub(">", "&gt;")
  s = s:gsub('"', "&quot;")
  s = s:gsub("'", "&apos;")
  return s
end

local function para_openxml_center(text, color_hex, bold, half_points, eastAsiaFont)
  local jc   = '<w:jc w:val="center"/>'
  local b    = bold and '<w:b/>' or ''
  local sz   = string.format('<w:sz w:val="%d"/>', half_points)
  local col  = string.format('<w:color w:val="%s"/>', color_hex)
  local font = ''
  if eastAsiaFont and eastAsiaFont ~= '' then
    font = string.format(
      '<w:rFonts w:ascii="%s" w:hAnsi="%s" w:eastAsia="%s"/>',
      eastAsiaFont, eastAsiaFont, eastAsiaFont
    )
  end

  -- 若文本含前后空格，需保留
  local t = string.format('<w:t xml:space="preserve">%s</w:t>', xml_escape(text))

  local xml = table.concat({
    '<w:p>',
      '<w:pPr>', jc, '</w:pPr>',
      '<w:r>',
        '<w:rPr>', font, col, b, sz, '</w:rPr>',
        t,
      '</w:r>',
    '</w:p>'
  })
  return pandoc.RawBlock('openxml', xml)
end

local function red_line_openxml()
  -- 上边框 2pt 的段落；w:sz 是半点/8，16≈2pt（可改 24 更粗）
  local xml = table.concat({
    '<w:p>',
      '<w:pPr>',
        '<w:pBdr>',
          '<w:top w:val="single" w:sz="16" w:space="0" w:color="D10000"/>',
        '</w:pBdr>',
        '<w:jc w:val="center"/>',
      '</w:pPr>',
      '<w:r><w:t xml:space="preserve"> </w:t></w:r>',
    '</w:p>'
  })
  return pandoc.RawBlock('openxml', xml)
end

function Div(el)
  -- 东亚字体名可按需填；留空则用 Word 默认
  local eastAsiaFont = 'SimHei'  -- 也可以用 'Microsoft YaHei'、'SimSun' 等

  if el.classes:includes('GovTitle') then
    local text = stringify_inlines(el.content)
    return para_openxml_center(text, 'D10000', true, 72, eastAsiaFont)   -- 36pt = 72 half-points
  end

  if el.classes:includes('GovDocNo') then
    local text = stringify_inlines(el.content)
    return para_openxml_center(text, '000000', false, 32, eastAsiaFont)  -- 16pt = 32 half-points
  end

  if el.classes:includes('GovRedLine') then
    return red_line_openxml()
  end

  return nil
end
