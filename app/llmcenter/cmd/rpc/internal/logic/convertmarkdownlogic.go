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

	// 1) 预处理 Markdown
	md := preprocessMarkdown(in.Markdown)

	// 2) 运行 pandoc
	outName := "export." + t
	contentType := map[string]string{
		"pdf":  "application/pdf",
		"docx": "application/vnd.openxmlformats-officedocument.wordprocessingml.document",
	}[t]

	data, err := runPandoc(l.ctx, md, t, l.svcCtx.Config.Font.Path)
	if err != nil {
		return nil, err
	}

	return &pb.ConvertMarkdownResponse{
		Filename:    outName,
		ContentType: contentType,
		Data:        data,
	}, nil
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
func runPandoc(ctx context.Context, markdown, typ, fontDir string) ([]byte, error) {
	// 1) 落地输入为临时 .md 文件
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

	// 2) 确定输出文件
	ext := "." + typ
	outFile := mdFile.Name() + ext
	defer os.Remove(outFile)

	// 3) 组装 pandoc 参数
	args := []string{
		"-f", "markdown",
		"-o", outFile,
		"--wrap=preserve",
		mdFile.Name(),
	}

	if typ == "pdf" {
		args = append([]string{"--pdf-engine=xelatex"}, args...)

		// 字体
		fontArgs := buildPandocFontArgs(fontDir)
		args = append(args, fontArgs...)

		// 页边距
		args = append(args, "-V", "geometry:top=20mm,left=20mm,right=20mm,bottom=20mm")
		// 取消首行缩进（可选）
		args = append(args, "-V", "indent=0")

		// ✅ 自动换行：调用 microtype 与 CJK 支持包
		args = append(args, "-V", "linestretch=1.2")
		args = append(args, "-V", "CJKmainfontoptions=AutoFakeBold,AutoFakeSlant")
		args = append(args, "-V", "header-includes=\\usepackage[slantfont,boldfont]{xeCJK}\\usepackage{microtype}\\tolerance=1000\\emergencystretch=3em\\sloppy")

		args = append(args, "--pdf-engine-opt=-halt-on-error", "--pdf-engine-opt=-interaction=nonstopmode")
	}

	// 4) 调用 pandoc
	// 加一个超时，防止卡死
	c, cancel := context.WithTimeout(ctx, 60*time.Second)
	defer cancel()

	cmd := exec.CommandContext(c, "pandoc", args...)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("pandoc 执行失败: %v\nstderr: %s", err, stderr.String())
	}

	// 5) 读回输出
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

/***************（保留以兼容 docx core 里可能用到的）***************/
func xmlEscape(s string) string { return html.EscapeString(s) }
