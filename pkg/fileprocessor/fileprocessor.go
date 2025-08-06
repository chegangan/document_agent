package fileprocessor

import (
	"archive/zip"
	"bytes"
	"encoding/xml"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/ledongthuc/pdf"
	"github.com/xuri/excelize/v2"
)

// ReadTextFile 读取文本文件内容
func ReadTextFile(filePath string) (string, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// ReadDocxFile 读取 docx 文件内容
func ReadDocxFile(filePath string) (string, error) {
	r, err := zip.OpenReader(filePath)
	if err != nil {
		return "", err
	}
	defer r.Close()

	var documentXML []byte
	for _, f := range r.File {
		if f.Name == "word/document.xml" {
			rc, err := f.Open()
			if err != nil {
				return "", err
			}
			defer rc.Close()

			documentXML, err = io.ReadAll(rc)
			if err != nil {
				return "", err
			}
			break
		}
	}

	if documentXML == nil {
		return "", fmt.Errorf("document.xml not found in docx")
	}

	type Text struct {
		XMLName xml.Name `xml:"t"`
		Content string   `xml:",chardata"`
	}

	decoder := xml.NewDecoder(bytes.NewReader(documentXML))
	var textBuilder strings.Builder

	for {
		tok, err := decoder.Token()
		if err == io.EOF {
			break
		} else if err != nil {
			return "", err
		}

		if se, ok := tok.(xml.StartElement); ok && se.Name.Local == "t" {
			var t Text
			if err := decoder.DecodeElement(&t, &se); err == nil {
				textBuilder.WriteString(t.Content)
			}
		}
	}

	return textBuilder.String(), nil
}

// ReadPdfFile 读取 pdf 文件内容并清理格式
func ReadPdfFile(filePath string) (string, error) {
	f, r, err := pdf.Open(filePath)
	if err != nil {
		return "", err
	}
	defer f.Close()

	b, err := r.GetPlainText()
	if err != nil {
		return "", err
	}

	rawText, err := io.ReadAll(b)
	if err != nil {
		return "", err
	}

	return cleanPdfText(string(rawText)), nil
}

// 读取 xlsx 文件的文本内容
func ReadXlsxFile(filePath string) (string, error) {
	f, err := excelize.OpenFile(filePath)
	if err != nil {
		return "", err
	}
	defer f.Close()

	var sb strings.Builder
	sheets := f.GetSheetList()
	for _, sheet := range sheets {
		rows, err := f.GetRows(sheet)
		if err != nil {
			continue
		}
		for _, row := range rows {
			sb.WriteString(strings.Join(row, "\t") + "\n")
		}
	}
	return sb.String(), nil
}

// 读取 pptx 文件内容
func ReadPptxFile(filePath string) (string, error) {
	r, err := zip.OpenReader(filePath)
	if err != nil {
		return "", err
	}
	defer r.Close()

	var sb strings.Builder

	for _, f := range r.File {
		if strings.HasPrefix(f.Name, "ppt/slides/slide") && strings.HasSuffix(f.Name, ".xml") {
			rc, err := f.Open()
			if err != nil {
				continue
			}
			defer rc.Close()

			content, err := extractTextFromSlideXML(rc)
			if err == nil {
				sb.WriteString(content + "\n")
			}
		}
	}

	return sb.String(), nil
}

func extractTextFromSlideXML(r io.Reader) (string, error) {
	decoder := xml.NewDecoder(r)
	var sb strings.Builder

	for {
		tok, err := decoder.Token()
		if err != nil {
			if err == io.EOF {
				break
			}
			return "", err
		}
		switch se := tok.(type) {
		case xml.StartElement:
			if se.Name.Local == "t" {
				var s struct {
					Text string `xml:",chardata"`
				}
				if err := decoder.DecodeElement(&s, &se); err == nil {
					sb.WriteString(s.Text)
				}
			}
		}
	}

	return sb.String(), nil
}

// cleanPdfText 去除 PDF 过多的换行符，只保留段落级别的换行
func cleanPdfText(raw string) string {
	lines := strings.Split(raw, "\n")
	var sb strings.Builder
	var paragraph strings.Builder

	for _, line := range lines {
		trimmedLine := strings.TrimSpace(line)
		if trimmedLine == "" {
			if paragraph.Len() > 0 {
				sb.WriteString(paragraph.String())
				sb.WriteString("\n\n") // 用双换行符分隔段落
				paragraph.Reset()
			}
			continue
		}
		// 将行连接起来，中间加一个空格
		if paragraph.Len() > 0 {
			paragraph.WriteString(" ")
		}
		paragraph.WriteString(trimmedLine)
	}

	if paragraph.Len() > 0 {
		sb.WriteString(paragraph.String())
	}

	return sb.String()
}
