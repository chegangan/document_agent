package logic

import (
	"archive/zip"
	"bytes"
	"context"
	"fmt"
	"html"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"document_agent/app/llmcenter/cmd/rpc/internal/svc"
	"document_agent/app/llmcenter/cmd/rpc/pb"
	"github.com/jung-kurt/gofpdf"
	"github.com/zeromicro/go-zero/core/logx"
	"golang.org/x/text/encoding/simplifiedchinese"
)

type ConvertMarkdownLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewConvertMarkdownLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ConvertMarkdownLogic {
	return &ConvertMarkdownLogic{ctx: ctx, svcCtx: svcCtx, Logger: logx.WithContext(ctx)}
}

func (l *ConvertMarkdownLogic) ConvertMarkdown(in *pb.ConvertMarkdownRequest) (*pb.ConvertMarkdownResponse, error) {
	t := strings.ToLower(strings.TrimSpace(in.Type))
	if t != "pdf" && t != "docx" {
		return nil, fmt.Errorf("type 仅支持 pdf 或 docx")
	}
	md := strings.ReplaceAll(in.Markdown, "\r\n", "\n")

	// 1) 解析 Markdown -> 极简 AST
	doc := parseMarkdown(md)

	// 2) 渲染
	switch t {
	case "pdf":
		data, err := renderPDF(doc, l.svcCtx.Config.Font.Path)
		if err != nil {
			return nil, fmt.Errorf("渲染 PDF 失败: %w", err)
		}
		return &pb.ConvertMarkdownResponse{
			Filename:    "export.pdf",
			ContentType: "application/pdf",
			Data:        data,
		}, nil
	case "docx":
		data, err := renderDOCX(doc)
		if err != nil {
			return nil, fmt.Errorf("渲染 DOCX 失败: %w", err)
		}
		return &pb.ConvertMarkdownResponse{
			Filename:    "export.docx",
			ContentType: "application/vnd.openxmlformats-officedocument.wordprocessingml.document",
			Data:        data,
		}, nil
	default:
		return nil, fmt.Errorf("不支持的 type: %s", t)
	}
}

/*************** Minimal Markdown AST ***************/

type NodeType int

const (
	NParagraph NodeType = iota
	NHeading
	NList
	NCodeBlock
)

type Inline struct {
	Text   string
	Bold   bool
	Italic bool
	Code   bool
}

type Block struct {
	Typ     NodeType
	Level   int        // heading level 1..6
	Ordered bool       // for list
	Items   [][]Inline // list items
	Lines   []string   // codeblock lines
	Inlines []Inline   // paragraph / heading
}

type Document struct{ Blocks []Block }

var (
	reFence  = regexp.MustCompile("^```")
	reH      = regexp.MustCompile(`^(#{1,6})\s+(.*)$`)
	reUItem  = regexp.MustCompile(`^\s*[-*+]\s+(.*)$`)
	reOItem  = regexp.MustCompile(`^\s*\d+\.\s+(.*)$`)
	reInline = regexp.MustCompile(`(\*\*[^*]+\*\*|\*[^*]+\*|` + "`" + `[^` + "`" + `]+` + "`" + `)`)
)

func parseMarkdown(md string) Document {
	lines := strings.Split(md, "\n")
	var blocks []Block
	var i int
	for i < len(lines) {
		line := lines[i]

		// fenced code block
		if reFence.MatchString(line) {
			i++
			var code []string
			for i < len(lines) && !reFence.MatchString(lines[i]) {
				code = append(code, lines[i])
				i++
			}
			if i < len(lines) && reFence.MatchString(lines[i]) {
				i++
			}
			blocks = append(blocks, Block{Typ: NCodeBlock, Lines: code})
			continue
		}

		// heading
		if m := reH.FindStringSubmatch(line); m != nil {
			level := len(m[1])
			inlines := parseInlines(m[2])
			blocks = append(blocks, Block{Typ: NHeading, Level: level, Inlines: inlines})
			i++
			continue
		}

		// list (ordered/unordered)
		if reUItem.MatchString(line) || reOItem.MatchString(line) {
			ordered := reOItem.MatchString(line)
			var items [][]Inline
			for i < len(lines) {
				if ordered && reOItem.MatchString(lines[i]) {
					items = append(items, parseInlines(strings.TrimSpace(splitAfter(lines[i], ". "))))
					i++
					continue
				}
				if !ordered && reUItem.MatchString(lines[i]) {
					// 去掉前缀符号
					li := reUItem.ReplaceAllString(lines[i], "$1")
					items = append(items, parseInlines(strings.TrimSpace(li)))
					i++
					continue
				}
				break
			}
			blocks = append(blocks, Block{Typ: NList, Ordered: ordered, Items: items})
			continue
		}

		// paragraph（合并连续非空行）
		if strings.TrimSpace(line) != "" {
			var buf []string
			for i < len(lines) && strings.TrimSpace(lines[i]) != "" {
				buf = append(buf, strings.TrimSpace(lines[i]))
				i++
			}
			text := strings.Join(buf, " ")
			blocks = append(blocks, Block{Typ: NParagraph, Inlines: parseInlines(text)})
			continue
		}
		i++
	}
	return Document{Blocks: blocks}
}

func parseInlines(s string) []Inline {
	if s == "" {
		return nil
	}
	var out []Inline
	idxs := reInline.FindAllStringIndex(s, -1)
	last := 0
	pushText := func(t string) {
		if t == "" {
			return
		}
		out = append(out, Inline{Text: t})
	}
	for _, idx := range idxs {
		if idx[0] > last {
			pushText(s[last:idx[0]])
		}
		token := s[idx[0]:idx[1]]
		switch {
		case strings.HasPrefix(token, "**"):
			out = append(out, Inline{Text: token[2 : len(token)-2], Bold: true})
		case strings.HasPrefix(token, "*"):
			out = append(out, Inline{Text: token[1 : len(token)-1], Italic: true})
		case strings.HasPrefix(token, "`"):
			out = append(out, Inline{Text: token[1 : len(token)-1], Code: true})
		}
		last = idx[1]
	}
	if last < len(s) {
		pushText(s[last:])
	}
	for i := range out {
		out[i].Text = strings.ReplaceAll(out[i].Text, "\t", "    ")
	}
	return out
}

func splitAfter(s, sep string) string {
	i := strings.Index(s, sep)
	if i < 0 {
		return s
	}
	return s[i+len(sep):]
}

/*************** PDF renderer ***************/

func renderPDF(doc Document, Path string) ([]byte, error) {
	pdf := gofpdf.New("P", "mm", "A4", "")
	// 注册中文字体
	pdf.AddUTF8Font("simhei", "", Path+"/DENG.TTF")
	pdf.AddUTF8Font("simhei", "B", Path+"/DENGB.TTF")
	pdf.SetMargins(20, 20, 20)
	pdf.AddPage()
	pdf.SetAutoPageBreak(true, 20)

	lineHeight := 6.0
	baseParaSize := 12.0

	bulletWidth := 6.0 // 列表前缀占位宽度

	for _, b := range doc.Blocks {
		switch b.Typ {
		case NHeading:
			size := 18.0 - float64(b.Level-1)*2
			if size < 12 {
				size = 12
			}
			// 标题统一中文字体；用 MultiCell 保证自动换行
			pdf.SetFont("simhei", "", size)
			pdf.MultiCell(0, lineHeight+2, joinInlines(b.Inlines), "", "L", false)
			pdf.Ln(1)

		case NParagraph:
			// 段落：合并 inline 后一次性 MultiCell，保证换行
			pdf.SetFont("simhei", "", baseParaSize)
			pdf.MultiCell(0, lineHeight, joinInlines(b.Inlines), "", "L", false)
			pdf.Ln(1)

		case NList:
			pdf.SetFont("simhei", "", baseParaSize)
			for i, item := range b.Items {
				// 计算前缀
				prefix := "• "
				if b.Ordered {
					prefix = fmt.Sprintf("%d. ", i+1)
				}

				// 记录当前坐标
				x := pdf.GetX()
				y := pdf.GetY()

				// 先画前缀小格
				pdf.CellFormat(bulletWidth, lineHeight, prefix, "", 0, "L", false, 0, "")

				// 同行从 prefix 右侧开始写主体，多行自动换行
				pdf.SetXY(x+bulletWidth, y)
				pdf.MultiCell(0, lineHeight, joinInlines(item), "", "L", false)
			}
			pdf.Ln(1)

		case NCodeBlock:
			// 代码块：同样用中文字体，防止中文字符出问题；用 MultiCell 自动换行
			pdf.SetFont("simhei", "", 11)
			for _, ln := range b.Lines {
				pdf.MultiCell(0, lineHeight, RepairChinese(ln), "", "L", false)
			}
			pdf.Ln(1)
		}
	}

	var buf bytes.Buffer
	if err := pdf.Output(&buf); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func joinInlines(in []Inline) string {
	var sb strings.Builder
	for _, x := range in {
		sb.WriteString(RepairChinese(x.Text))
	}
	return sb.String()
}

func RepairChinese(s string) string {
	if s == "" {
		return s
	}
	// 先尝试：把每个 rune 当 1 字节还原，再按 UTF-8 验证
	if b, ok := latin1BytesFromRunes(s); ok && utf8.Valid(b) {
		return string(b)
	}
	// 再尝试：还原后按 GB18030 解
	if b, ok := latin1BytesFromRunes(s); ok {
		if out, err := simplifiedchinese.GB18030.NewDecoder().Bytes(b); err == nil && utf8.Valid(out) {
			return string(out)
		}
	}
	// 都不匹配就原样返回
	return s
}

func latin1BytesFromRunes(s string) ([]byte, bool) {
	out := make([]byte, 0, len(s))
	for _, r := range s {
		if r > 0xFF { // 正常中文会触发这里 -> 放弃本轮修复
			return nil, false
		}
		out = append(out, byte(r))
	}
	return out, true
}

/*************** DOCX renderer (handcrafted OpenXML) ***************/

type fileMap map[string][]byte

func (fm fileMap) add(name string, data []byte) { fm[name] = data }

func (fm fileMap) zip() ([]byte, error) {
	var names []string
	for k := range fm {
		names = append(names, k)
	}
	sort.Strings(names)

	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	for _, name := range names {
		w, err := zw.Create(name)
		if err != nil {
			return nil, err
		}
		if _, err = w.Write(fm[name]); err != nil {
			return nil, err
		}
	}
	if err := zw.Close(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func renderDOCX(doc Document) ([]byte, error) {
	fm := fileMap{}
	fm.add("[Content_Types].xml", []byte(contentTypesXML()))
	fm.add("_rels/.rels", []byte(relsXML()))
	fm.add("docProps/app.xml", []byte(appXML()))
	fm.add("docProps/core.xml", []byte(coreXML()))
	fm.add("word/document.xml", []byte(buildWordDocumentXML(doc)))
	fm.add("word/_rels/document.xml.rels", []byte(docRelsXML()))
	return fm.zip()
}

const xmlHeader = `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>`

func buildWordDocumentXML(doc Document) string {
	var b strings.Builder
	// 使用最小命名空间集合：w + r
	b.WriteString(xmlHeader + `<w:document xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main" xmlns:r="http://schemas.openxmlformats.org/officeDocument/2006/relationships"><w:body>`)

	for _, bl := range doc.Blocks {
		switch bl.Typ {
		case NHeading:
			size := 32 - (bl.Level-1)*3 // half-points
			if size < 18 {
				size = 18
			}
			b.WriteString(paragraphXML(bl.Inlines, size, true))
		case NParagraph:
			b.WriteString(paragraphXML(bl.Inlines, 24, false)) // 12pt
		case NList:
			for i, it := range bl.Items {
				prefix := "• "
				if bl.Ordered {
					prefix = strconv.Itoa(i+1) + ". "
				}
				li := append([]Inline{{Text: prefix, Bold: true}}, it...)
				b.WriteString(paragraphXML(li, 24, false))
			}
		case NCodeBlock:
			var in []Inline
			for _, ln := range bl.Lines {
				in = append(in, Inline{Text: ln})
				in = append(in, Inline{Text: "\n"})
			}
			b.WriteString(paragraphXMLCode(in))
		}
	}

	// 页面设置（A4, 边距 2.54cm）
	b.WriteString(`<w:sectPr><w:pgSz w:w="11906" w:h="16838"/><w:pgMar w:top="1440" w:right="1440" w:bottom="1440" w:left="1440" w:header="708" w:footer="708" w:gutter="0"/></w:sectPr>`)
	b.WriteString(`</w:body></w:document>`)
	return b.String()
}

func paragraphXML(in []Inline, size int, heading bool) string {
	var sb strings.Builder
	sb.WriteString(`<w:p><w:rPr>`)
	if heading {
		sb.WriteString(`<w:b/>`)
	}
	sb.WriteString(`</w:rPr>`)
	for _, r := range in {
		sb.WriteString(`<w:r><w:rPr>`)
		if r.Bold {
			sb.WriteString(`<w:b/>`)
		}
		if r.Italic {
			sb.WriteString(`<w:i/>`)
		}
		if r.Code {
			sb.WriteString(`<w:rFonts w:ascii="Courier New" w:hAnsi="Courier New"/>`)
		}
		sb.WriteString(`<w:sz w:val="` + strconv.Itoa(size) + `"/></w:rPr><w:t xml:space="preserve">` + xmlEscape(r.Text) + `</w:t></w:r>`)
	}
	sb.WriteString(`</w:p>`)
	return sb.String()
}

func paragraphXMLCode(in []Inline) string {
	var sb strings.Builder
	sb.WriteString(`<w:p><w:r><w:rPr><w:rFonts w:ascii="Courier New" w:hAnsi="Courier New"/><w:sz w:val="22"/></w:rPr><w:t xml:space="preserve">`)
	sb.WriteString(xmlEscape(joinInlines(in)))
	sb.WriteString(`</w:t></w:r></w:p>`)
	return sb.String()
}

func xmlEscape(s string) string { return html.EscapeString(s) }

func contentTypesXML() string {
	return xmlHeader + `
<Types xmlns="http://schemas.openxmlformats.org/package/2006/content-types">
  <Default Extension="rels" ContentType="application/vnd.openxmlformats-package.relationships+xml"/>
  <Default Extension="xml" ContentType="application/xml"/>
  <Override PartName="/word/document.xml" ContentType="application/vnd.openxmlformats-officedocument.wordprocessingml.document.main+xml"/>
  <Override PartName="/docProps/core.xml" ContentType="application/vnd.openxmlformats-package.core-properties+xml"/>
  <Override PartName="/docProps/app.xml" ContentType="application/vnd.openxmlformats-officedocument.extended-properties+xml"/>
</Types>`
}

func relsXML() string {
	return xmlHeader + `
<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">
  <Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/officeDocument" Target="word/document.xml"/>
</Relationships>`
}

func docRelsXML() string {
	return xmlHeader + `
<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">
</Relationships>`
}

func appXML() string {
	return xmlHeader + `
<Properties xmlns="http://schemas.openxmlformats.org/office/2006/extended-properties"
 xmlns:vt="http://schemas.openxmlformats.org/office/2006/docPropsVTypes">
  <Application>go-zero md2docx</Application>
</Properties>`
}

func coreXML() string {
	now := time.Now().UTC().Format(time.RFC3339)
	return xmlHeader + `
<cp:coreProperties xmlns:cp="http://schemas.openxmlformats.org/package/2006/metadata/core-properties"
 xmlns:dc="http://purl.org/dc/elements/1.1/"
 xmlns:dcterms="http://purl.org/dc/terms/"
 xmlns:dcmitype="http://purl.org/dc/dcmitype/"
 xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance">
  <dc:title>export</dc:title>
  <dc:creator>md-converter</dc:creator>
  <cp:lastModifiedBy>md-converter</cp:lastModifiedBy>
  <dcterms:created xsi:type="dcterms:W3CDTF">` + now + `</dcterms:created>
  <dcterms:modified xsi:type="dcterms:W3CDTF">` + now + `</dcterms:modified>
</cp:coreProperties>`
}
