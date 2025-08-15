package logic

import (
	"bytes"
	"context"
	"fmt"
	"html"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"

	"document_agent/app/llmcenter/cmd/rpc/internal/svc"
	"document_agent/app/llmcenter/cmd/rpc/pb"
	"github.com/zeromicro/go-zero/core/logx"
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

	// 1) 预处理 Markdown// 1) 预处理
	md := preprocessMarkdown(in.Markdown)

	// 2) 应用“首行居中、末两行右对齐”
	md = applyLineAlignments(md)

	title, docNo := pickTitleDocNo(in.GetInformation())

	if title == "" {
		title = "某某县人民政府文件" // 默认值
	}
	if docNo == "" {
		docNo = "某政【2025】1号" // 默认值
	}

	md = decorateGovHeaderAndBody(md, t, title, docNo)

	// 3) 运行 pandoc
	outName := "export." + t
	contentType := map[string]string{
		"pdf":  "application/pdf",
		"docx": "application/vnd.openxmlformats-officedocument.wordprocessingml.document",
	}[t]

	data, err := runPandoc(
		l.ctx,
		md,
		t,
		l.svcCtx.Config.Font.Path,
		l.svcCtx.Config.LuaFilters.Align,
		l.svcCtx.Config.LuaFilters.Gov,
		title, // ✅ 传入
		docNo, // ✅ 传入
	)

	if err != nil {
		return nil, err
	}

	return &pb.ConvertMarkdownResponse{
		Filename:    outName,
		ContentType: contentType,
		Data:        data,
	}, nil
}

// 根据输出类型，拼接“红字抬头 + 文号 + 红线”，并把正文首/末行对齐
func decorateGovHeaderAndBody(src, typ, title, docNo string) string {
	header := ""
	switch typ {
	case "pdf":
		// 不在 Markdown 里塞 LaTeX 抬头！交给 runPandoc 用 include-before-body 注入
		header = "" // 保持为空
	case "docx":
		header = fmt.Sprintf(`
::: {.GovTitle}
%s
:::

::: {.GovDocNo}
%s
:::

::: {.GovRedLine}
 
:::
`, html.EscapeString(title), html.EscapeString(docNo))
	}
	return header + "\n" + src
}

/*************** 预处理 ***************/

// 1) 把字面量的 "\n" 变成真正换行
// 2) 把 1.2.3. 之类的编号里的 "." 全部转义为 "\."（避开代码块和行内代码）
func preprocessMarkdown(src string) string {
	if src == "" {
		return src
	}

	// 把 CRLF 归一化（可选）
	s := strings.ReplaceAll(src, "\r\n", "\n")

	// 把字面量的 \n（两个字符：反斜杠 + n）转换为真正换行
	s = strings.ReplaceAll(s, `\n`, "\n")

	// 转义编号里的点；需要跳过 fenced code 与行内 code
	return escapeNumberDotsOutsideCode(s)
}

// 在代码块 (``` ... ```) 与行内代码 (`...`) 外部，把形如 1.2.3. 的片段里的 '.' 全部转义为 '\.'
// 例： "章节 1.2.3. 概述" -> "章节 1\.2\.3\. 概述"
func escapeNumberDotsOutsideCode(s string) string {
	// 状态机：fenced code、inline code、plain
	var (
		out      strings.Builder
		i        int
		inFence  bool
		inInline bool
		lines    = strings.Split(s, "\n")
	)

	for _, line := range lines {
		ln := line

		// fenced code block 的开始/结束：以 ``` 开头
		trim := strings.TrimSpace(ln)
		if strings.HasPrefix(trim, "```") {
			inFence = !inFence
			out.WriteString(ln)
			out.WriteByte('\n')
			continue
		}
		if inFence {
			// 代码块内不处理
			out.WriteString(ln)
			out.WriteByte('\n')
			continue
		}

		// 行内 code：使用一个简单扫描，遇到 ` 切换 inInline 状态
		var buf strings.Builder
		for i = 0; i < len(ln); i++ {
			if ln[i] == '`' {
				inInline = !inInline
				buf.WriteByte('`')
				continue
			}
			if inInline {
				buf.WriteByte(ln[i])
				continue
			}
			// 非代码区域：在这里做 1.2.3. 的点转义
			buf.WriteByte(ln[i])
		}
		processed := escapeNumberDots(buf.String())
		out.WriteString(processed)
		out.WriteByte('\n')
	}

	return out.String()
}

// 把行内的 1.2.3.（一个或多个 . 连接的数字段）整体替换
// 将其中所有 "." 换成 "\."；例如 10.01.2 -> 10\.01\.2
// 使用 \b 边界减少误伤，但依然会作用于 IP/版本号等（按你的需求，这样是预期的）
func escapeNumberDots(line string) string {
	var out strings.Builder
	i := 0
	for i < len(line) {
		// 收集一段 [0-9.] 串
		j := i
		for j < len(line) {
			c := line[j]
			if (c >= '0' && c <= '9') || c == '.' {
				j++
			} else {
				break
			}
		}
		if j == i { // 当前不是数字或点
			out.WriteByte(line[i])
			i++
			continue
		}

		token := line[i:j]
		// 规则：token 内必须包含至少一个 '.'，且不能全是点，且不能包含字母（已保证）
		if strings.Contains(token, ".") && token != "." {
			// 只要是纯数字点的组合，我们认为是编号串，全部 '.' 转义
			token = strings.ReplaceAll(token, ".", `\.`)
		}
		out.WriteString(token)
		i = j
	}
	return out.String()
}

/*************** 调用 Pandoc ***************/

// 通过 pandoc 把 markdown 渲染为指定类型 (pdf/docx)
func runPandoc(ctx context.Context, markdown, typ, fontDir, alignLua, govLua string, pdfTitle, pdfDocNo string) ([]byte, error) {
	mdFile, err := os.CreateTemp("", "md2-"+time.Now().Format("20060102150405")+"-*.md")
	if err != nil {
		return nil, fmt.Errorf("创建临时文件失败: %w", err)
	}
	defer os.Remove(mdFile.Name())
	defer mdFile.Close()
	if _, err := mdFile.WriteString(markdown); err != nil {
		return nil, fmt.Errorf("写入临时 Markdown 失败: %w", err)
	}
	_ = mdFile.Close()

	outFile := mdFile.Name() + "." + typ
	defer os.Remove(outFile)

	fromFmt := "markdown+fenced_divs"
	if typ == "docx" {
		fromFmt = "markdown+fenced_divs"
	} else if typ == "pdf" {
		fromFmt = "markdown+fenced_divs" // 这里不用 raw_tex 了，抬头走 include-before-body 更稳
	}

	args := []string{
		"-f", fromFmt,
		"-o", outFile,
		"--wrap=preserve",
	}

	// 如果是 PDF，注入一个真正的 LaTeX 头（红字抬头 + 文号 + 红线）
	var incFile string
	if typ == "pdf" {
		args = append([]string{"--pdf-engine=xelatex"}, args...)
		fontArgs := buildPandocFontArgs(fontDir)
		args = append(args, fontArgs...)
		args = append(args, "-V", "geometry:top=20mm,left=20mm,right=20mm,bottom=20mm")
		args = append(args, "-V", "indent=0")
		args = append(args, "-V", "linestretch=1.2")
		args = append(args, "-V", "CJKmainfontoptions=AutoFakeBold,AutoFakeSlant")
		args = append(args, "-V",
			`header-includes=\usepackage[slantfont,boldfont]{xeCJK}\usepackage{microtype}\usepackage{xcolor}\tolerance=1000\emergencystretch=3em\sloppy`)
		args = append(args, "--pdf-engine-opt=-halt-on-error", "--pdf-engine-opt=-interaction=nonstopmode")

		// 这里写 include-before-body 内容（注意花括号作用域，\centering 不外溢）
		tex := buildPdfHeaderTex(pdfTitle, pdfDocNo)

		f, e := os.CreateTemp("", "inc-*.tex")
		if e != nil {
			return nil, fmt.Errorf("创建临时 tex 失败: %w", e)
		}
		incFile = f.Name()
		if _, e := f.WriteString(tex); e != nil {
			f.Close()
			return nil, fmt.Errorf("写入临时 tex 失败: %w", e)
		}
		f.Close()
		defer os.Remove(incFile)

		args = append(args, "--include-before-body="+incFile)
		args = append(args, "--lua-filter="+alignLua)
	} else if typ == "docx" {
		args = append(args, "--lua-filter="+alignLua)
		args = append(args, "--lua-filter="+govLua)
	}

	args = append(args, mdFile.Name())

	c, cancel := context.WithTimeout(ctx, 60*time.Second)
	defer cancel()
	cmd := exec.CommandContext(c, "pandoc", args...)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("pandoc 执行失败: %v\nstderr: %s", err, stderr.String())
	}
	data, err := os.ReadFile(outFile)
	if err != nil {
		return nil, fmt.Errorf("读取输出文件失败: %w", err)
	}
	return data, nil
}

// 构建 pandoc 字体参数（PDF 用）
func buildPandocFontArgs(fontDir string) []string {
	// 统一正斜杠 & 结尾加 "/"
	normDir := fontDir
	normDir = strings.ReplaceAll(normDir, "\\", "/")
	if normDir != "" && !strings.HasSuffix(normDir, "/") {
		normDir += "/"
	}

	// 优先尝试 DENG / DENGB 组合
	if normDir != "" {
		if _, err1 := os.Stat(normDir + "DENG.TTF"); err1 == nil {
			args := []string{
				"-V", "mainfont=DENG",
				"-V", "mainfontoptions=Path=" + normDir + ",Extension=.TTF",
			}
			if _, err2 := os.Stat(normDir + "DENGB.TTF"); err2 == nil {
				// 有粗体文件就一起指定
				args[3] = args[3] + ",BoldFont=DENGB.TTF"
			}
			return args
		}
		// 其次尝试 SIMHEI（常见中文黑体，可能没有独立粗体文件）
		if _, err := os.Stat(normDir + "SIMHEI.TTF"); err == nil {
			return []string{
				"-V", "mainfont=SIMHEI",
				"-V", "mainfontoptions=Path=" + normDir + ",Extension=.TTF",
			}
		}
	}

	// 回退：系统字体名（不带路径），避免 LaTeX 解析路径
	switch runtime.GOOS {
	case "windows":
		return []string{"-V", "mainfont=Microsoft YaHei"}
	case "darwin":
		return []string{"-V", "mainfont=PingFang SC"}
	default:
		return []string{"-V", "mainfont=Noto Sans CJK SC"}
	}
}

func applyLineAlignments(src string) string {
	if src == "" {
		return src
	}
	lines := strings.Split(src, "\n")

	// 找出所有非空行的下标
	nonEmpty := make([]int, 0, len(lines))
	for i, ln := range lines {
		if strings.TrimSpace(ln) != "" {
			nonEmpty = append(nonEmpty, i)
		}
	}
	if len(nonEmpty) == 0 {
		return src
	}

	// 第一行 -> 居中
	first := nonEmpty[0]
	lines[first] = wrapAlignDiv(lines[first], "center")

	// 最后两行 -> 右对齐
	if len(nonEmpty) >= 2 {
		last := nonEmpty[len(nonEmpty)-1]
		lines[last] = wrapAlignDiv(lines[last], "right")
	}
	if len(nonEmpty) >= 3 {
		secondLast := nonEmpty[len(nonEmpty)-2]
		lines[secondLast] = wrapAlignDiv(lines[secondLast], "right")
	}

	return strings.Join(lines, "\n")
}

func wrapAlignDiv(line, align string) string {
	// 已经是 fenced div 且含 align 就不重复包裹
	trimmed := strings.TrimSpace(line)
	low := strings.ToLower(trimmed)
	if strings.HasPrefix(low, ":::") && strings.Contains(low, "align=") {
		return line
	}
	// Pandoc 原生 fenced div，docx writer 能正确识别 {align=...}
	// 形如：
	// ::: {align=center}
	// 第一行文本
	// :::
	return fmt.Sprintf("::: {align=%s}\n%s\n:::", align, line)
}

// 从 repeated InfoItem 中提取 Title / DocNo（大小写不敏感），空白会被 Trim
func pickTitleDocNo(items []*pb.InfoItem) (title, docNo string) {
	for _, it := range items {
		t := strings.ToLower(strings.TrimSpace(it.GetType()))
		v := strings.TrimSpace(it.GetContant())
		if v == "" {
			continue
		}
		switch t {
		case "title":
			title = v
		case "docno":
			docNo = v
		}
	}
	return
}

// 新增：根据 title/docNo 生成 tex 头
func buildPdfHeaderTex(title, docNo string) string {
	esc := func(s string) string {
		// 极简 LaTeX 转义（足够覆盖常见中文标题中的特殊字符）
		replacer := strings.NewReplacer(
			`%`, `\%`, `$`, `\$`, `#`, `\#`, `&`, `\&`,
			`_`, `\_`, `{`, `\{`, `}`, `\}`, `^`, `\^{}`, `~`, `\~{}`,
		)
		return replacer.Replace(s)
	}
	return fmt.Sprintf(
		`{\centering {\fontsize{36pt}{42pt}\selectfont\textcolor{red}{%s}}\par}
\vspace{4pt}
{\centering {\large %s}\par}
{\color{red}\rule{\linewidth}{1.2pt}}
\vspace{8pt}
`, esc(title), esc(docNo))
}

/***************（保留以兼容 docx core 里可能用到的）***************/
func xmlEscape(s string) string { return html.EscapeString(s) }
