package handlers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"html/template"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"anan-diary/ad-backend/internal/repository"
	"anan-diary/ad-backend/pkg/middleware"
	"anan-diary/ad-backend/pkg/response"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type ExportHandler struct {
	recordRepo *repository.GrowthRecordRepository
	babyRepo   *repository.BabyRepository
	memberRepo *repository.FamilyMemberRepository
	logger     *zap.Logger
	loc        *time.Location
}

func NewExportHandler(
	recordRepo *repository.GrowthRecordRepository,
	babyRepo *repository.BabyRepository,
	memberRepo *repository.FamilyMemberRepository,
	logger *zap.Logger,
) *ExportHandler {
	// 加载 Asia/Shanghai 时区
	loc, err := time.LoadLocation("Asia/Shanghai")
	if err != nil {
		logger.Fatal("Failed to load Asia/Shanghai timezone", zap.Error(err))
	}
	return &ExportHandler{
		recordRepo: recordRepo,
		babyRepo:   babyRepo,
		memberRepo: memberRepo,
		logger:     logger,
		loc:        loc,
	}
}

// MediaItem 媒体文件项
type ExportMediaItem struct {
	Type      string `json:"type"`
	URL       string `json:"url"`
	LocalPath string `json:"local_path"`
}

// toShanghaiTime 将 GORM 读取的时间（被误认为 UTC）转换为上海时间（UTC+8）
// 数据库中的时间实际上是上海时间，但 GORM 读取 timestamp without time zone 时视为 UTC
func toShanghaiTime(t time.Time) time.Time {
	shanghai := time.FixedZone("CST", 8*3600)
	return time.Date(t.Year(), t.Month(), t.Day(),
		t.Hour(), t.Minute(), t.Second(), t.Nanosecond(), shanghai)
}

// ExportRecord 导出记录项
type ExportRecord struct {
	ID         uint              `json:"id"`
	BabyID     uint              `json:"baby_id"`
	UserID     uint              `json:"user_id"`
	Content    string            `json:"content"`
	Tags       []string          `json:"tags"`
	Media      []ExportMediaItem `json:"media"`
	CreatedAt  time.Time         `json:"created_at"`
	MemberName string            `json:"member_name"`
	MemberRole string            `json:"member_role"`
}

// PDFData 用于 HTML 模板的数据
type PDFData struct {
	BabyName    string
	BirthDate   string
	AgeDays     int
	RecordCount int
	ExportTime  string
	Records     []ExportRecord
}

// generateHTML 生成 PDF 的 HTML 内容
func (h *ExportHandler) generateHTML(data PDFData) (string, error) {
	htmlTemplate := `<!DOCTYPE html>
<html lang="zh-CN">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>{{.BabyName}}的成长日记</title>
    <style>
        @page {
            size: A4;
            margin: 20mm;
        }
        * {
            margin: 0;
            padding: 0;
            box-sizing: border-box;
        }
        body {
            font-family: "Noto Sans CJK SC", "Source Han Sans SC", "Microsoft YaHei", "SimHei", sans-serif;
            line-height: 1.8;
            color: #333;
        }

        /* 封面页 */
        .cover-page {
            page-break-after: always;
            height: 297mm;
            display: flex;
            flex-direction: column;
            justify-content: center;
            align-items: center;
            text-align: center;
            background: linear-gradient(135deg, #fff5f5 0%, #fff 50%, #f5f8ff 100%);
        }
        .cover-title {
            font-size: 36px;
            font-weight: bold;
            color: #ff6b8a;
            margin-bottom: 30px;
        }
        .cover-subtitle {
            font-size: 18px;
            color: #888;
            margin-bottom: 60px;
        }
        .cover-info {
            background: white;
            padding: 40px 60px;
            border-radius: 20px;
            box-shadow: 0 10px 40px rgba(0,0,0,0.1);
        }
        .cover-info-item {
            font-size: 16px;
            margin: 15px 0;
            color: #555;
        }
        .cover-info-item label {
            font-weight: bold;
            color: #333;
        }
        .cover-footer {
            position: absolute;
            bottom: 40px;
            font-size: 14px;
            color: #999;
        }

        /* 记录页 */
        .record-page {
            page-break-after: always;
            min-height: 297mm;
            padding-top: 20px;
        }
        .page-header {
            display: flex;
            justify-content: space-between;
            align-items: center;
            padding: 15px 0;
            border-bottom: 2px solid #ffb6c1;
            margin-bottom: 25px;
        }
        .page-header-title {
            font-size: 14px;
            color: #ff6b8a;
            font-weight: bold;
        }
        .page-header-page {
            font-size: 12px;
            color: #999;
        }

        /* 记录项 */
        .record-item {
            background: #fff;
            border-radius: 12px;
            padding: 20px;
            margin-bottom: 20px;
            border: 1px solid #ffe0e6;
            box-shadow: 0 2px 8px rgba(255,107,138,0.08);
        }
        .record-header {
            display: flex;
            justify-content: space-between;
            align-items: center;
            margin-bottom: 12px;
        }
        .record-date {
            font-size: 14px;
            color: #ff6b8a;
            font-weight: bold;
        }
        .record-author {
            font-size: 13px;
            color: #888;
        }
        .record-tags {
            display: flex;
            gap: 8px;
            margin-bottom: 12px;
        }
        .tag {
            display: inline-block;
            background: linear-gradient(135deg, #ffe0e6 0%, #ffc4d4 100%);
            color: #ff6b8a;
            padding: 3px 10px;
            border-radius: 12px;
            font-size: 12px;
        }
        .record-content {
            font-size: 15px;
            line-height: 1.9;
            color: #333;
            white-space: pre-wrap;
            margin-bottom: 15px;
        }
        .record-media {
            display: flex;
            flex-wrap: wrap;
            gap: 10px;
        }
        .media-item {
            max-width: 200px;
            border-radius: 8px;
            overflow: hidden;
            box-shadow: 0 2px 6px rgba(0,0,0,0.1);
        }
        .media-item img {
            display: block;
            width: 100%;
            height: auto;
        }
        .media-placeholder {
            display: inline-block;
            background: #f5f5f5;
            color: #888;
            padding: 8px 15px;
            border-radius: 6px;
            font-size: 13px;
        }

        /* 分隔线 */
        .record-separator {
            border: none;
            border-top: 1px dashed #ffd1dc;
            margin: 20px 0;
        }

        /* 封底页 */
        .back-page {
            page-break-before: always;
            height: 297mm;
            display: flex;
            flex-direction: column;
            justify-content: center;
            align-items: center;
            text-align: center;
            background: linear-gradient(135deg, #fff5f5 0%, #fff 50%, #f5f8ff 100%);
        }
        .back-message {
            font-size: 20px;
            color: #666;
            margin: 20px 0;
        }
        .back-blessing {
            font-size: 24px;
            color: #ff6b8a;
            font-weight: bold;
            margin-top: 30px;
        }

        /* 打印优化 */
        @media print {
            body {
                -webkit-print-color-adjust: exact;
                print-color-adjust: exact;
            }
        }
    </style>
</head>
<body>
    <!-- 封面页 -->
    <div class="cover-page">
        <div class="cover-title">{{.BabyName}}的成长日记</div>
        <div class="cover-subtitle">记录从出生到现在的点点滴滴</div>
        <div class="cover-info">
            <div class="cover-info-item"><label>宝宝姓名：</label>{{.BabyName}}</div>
            <div class="cover-info-item"><label>出生日期：</label>{{.BirthDate}}</div>
            <div class="cover-info-item"><label>当前年龄：</label>{{.AgeDays}} 天</div>
            <div class="cover-info-item"><label>记录总数：</label>{{.RecordCount}} 条</div>
        </div>
        <div class="cover-footer">导出时间：{{.ExportTime}}</div>
    </div>

    <!-- 记录页 -->
    {{range $pageIndex, $pageRecords := splitRecords .Records 5}}
    <div class="record-page">
        <div class="page-header">
            <span class="page-header-title">{{$.BabyName}}的成长日记</span>
            <span class="page-header-page">第{{add $pageIndex 1}}页</span>
        </div>

        {{range $index, $record := $pageRecords}}
        <div class="record-item">
            <div class="record-header">
                <span class="record-date">[{{$record.CreatedAt | formatDate}}]</span>
                {{if $record.MemberName}}
                <span class="record-author">· {{$record.MemberRole}}({{$record.MemberName}})</span>
                {{end}}
            </div>

            {{if $record.Tags}}
            <div class="record-tags">
                {{range $record.Tags}}
                <span class="tag">{{.}}</span>
                {{end}}
            </div>
            {{end}}

            <div class="record-content">{{$record.Content}}</div>

            {{if $record.Media}}
            <div class="record-media">
                {{range $record.Media}}
                {{if eq .Type "image"}}
                <div class="media-item">
                    <img src="{{.LocalPath}}" alt="图片" />
                </div>
                {{else}}
                <span class="media-placeholder">📹 视频</span>
                {{end}}
                {{end}}
            </div>
            {{end}}
        </div>

        {{if not $index}}
        <hr class="record-separator">
        {{end}}
        {{end}}
    </div>
    {{end}}

    <!-- 封底页 -->
    <div class="back-page">
        <div class="back-message">感谢记录宝宝成长的每一个瞬间</div>
        <div class="back-blessing">愿宝宝健康快乐地长大</div>
    </div>
</body>
</html>`

	tmpl, err := template.New("pdf").Funcs(template.FuncMap{
		"add": func(a, b int) int { return a + b },
		"formatDate": func(t time.Time) string {
			return toShanghaiTime(t).Format("2006-01-02 15:04")
		},
		"splitRecords": func(records []ExportRecord, size int) [][]ExportRecord {
			if len(records) == 0 {
				return [][]ExportRecord{}
			}
			var chunks [][]ExportRecord
			for i := 0; i < len(records); i += size {
				end := i + size
				if end > len(records) {
					end = len(records)
				}
				chunks = append(chunks, records[i:end])
			}
			return chunks
		},
	}).Parse(htmlTemplate)
	if err != nil {
		return "", err
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", err
	}

	return buf.String(), nil
}

// ExportToPDF 导出宝宝日记为 PDF（使用 Chrome Headless）
func (h *ExportHandler) ExportToPDF(c *gin.Context) {
	userID, exists := middleware.GetUserID(c)
	if !exists {
		response.Unauthorized(c, "未登录")
		return
	}

	babyIDStr := c.Query("baby_id")
	babyID, err := strconv.ParseUint(babyIDStr, 10, 32)
	if err != nil {
		response.ValidationError(c, "无效的 baby_id")
		return
	}

	// 获取宝宝信息
	baby, err := h.babyRepo.GetByID(uint(babyID))
	if err != nil {
		h.logger.Error("Failed to get baby", zap.Error(err))
		response.NotFound(c, "宝宝档案不存在")
		return
	}

	// 获取宝宝的所有成长记录
	records, err := h.recordRepo.GetByBabyID(uint(babyID))
	if err != nil {
		h.logger.Error("Failed to get records", zap.Error(err))
		response.ServerError(c, "获取记录失败")
		return
	}

	// 获取宝宝的所有家庭成员
	members, err := h.memberRepo.GetByBabyID(uint(babyID))
	if err != nil {
		h.logger.Error("Failed to get family members", zap.Error(err))
	}

	// 转换为导出格式
	exportRecords := make([]ExportRecord, 0, len(records))
	for _, r := range records {
		memberName := ""
		memberRole := "其他"
		for _, m := range members {
			if r.UserID != 0 && m.UserID != nil && r.UserID == *m.UserID {
				memberName = m.Name
				memberRole = m.Role
				break
			}
		}

		var tags []string
		if r.Tags != "" && r.Tags != "[]" {
			json.Unmarshal([]byte(r.Tags), &tags)
		}

		var mediaItems []ExportMediaItem
		if r.Media != "" && r.Media != "[]" {
			json.Unmarshal([]byte(r.Media), &mediaItems)
			// 转换为本地路径
			for i := range mediaItems {
				if mediaItems[i].Type == "image" && mediaItems[i].URL != "" {
					// 从 URL 提取文件名
					filename := getImageFileName(mediaItems[i].URL)
					// 构建本地路径
					localPath := filepath.Join("uploads/records/images", filename)
					// 检查文件是否存在
					if _, err := os.Stat(localPath); err == nil {
						// 转换为绝对路径
						absPath, err := filepath.Abs(localPath)
						if err == nil {
							mediaItems[i].LocalPath = absPath
						} else {
							mediaItems[i].LocalPath = localPath
						}
					}
				}
			}
		}

		exportRecords = append(exportRecords, ExportRecord{
			ID:         r.ID,
			BabyID:     r.BabyID,
			UserID:     r.UserID,
			Content:    r.Content,
			Tags:       tags,
			Media:      mediaItems,
			CreatedAt:  toShanghaiTime(r.CreatedAt),
			MemberName: memberName,
			MemberRole: memberRole,
		})
	}

	// 按时间排序（从早到晚）
	sort.Slice(exportRecords, func(i, j int) bool {
		return exportRecords[i].CreatedAt.Before(exportRecords[j].CreatedAt)
	})

	// 计算年龄
	var birthDate time.Time
	// 尝试多种格式解析出生日期
	birthDate, err = time.Parse("2006-01-02 15:04:05", baby.BirthDate)
	if err != nil {
		birthDate, err = time.Parse("2006-01-02", baby.BirthDate)
	}
	if err != nil {
		h.logger.Warn("Failed to parse birth date", zap.String("birth_date", baby.BirthDate), zap.Error(err))
		// 如果解析失败，尝试直接使用字符串的前 10 个字符
		birthDate, _ = time.Parse("2006-01-02", baby.BirthDate[:10])
	}
	today := time.Now()
	ageDays := int(today.Sub(birthDate).Hours() / 24)
	h.logger.Info("Age calculated", zap.String("birth_date", baby.BirthDate), zap.Time("parsed_birth_date", birthDate), zap.Int("age_days", ageDays))

	// 准备模板数据
	pdfData := PDFData{
		BabyName:    baby.Name,
		BirthDate:   baby.BirthDate,
		AgeDays:     ageDays,
		RecordCount: len(exportRecords),
		ExportTime:  today.Format("2006 年 01 月 02 日 15:04"),
		Records:     exportRecords,
	}

	// 生成 HTML
	htmlContent, err := h.generateHTML(pdfData)
	if err != nil {
		h.logger.Error("Failed to generate HTML", zap.Error(err))
		response.ServerError(c, "生成 PDF 内容失败")
		return
	}

	// 创建临时文件
	tmpDir := os.TempDir()
	htmlPattern := fmt.Sprintf("baby-diary-%d-*.html", userID)
	pdfPattern := fmt.Sprintf("baby-diary-%d-*.pdf", userID)

	// 写入 HTML 文件
	htmlPath, err := writeTempFile(filepath.Join(tmpDir, htmlPattern), htmlContent)
	if err != nil {
		h.logger.Error("Failed to write HTML file", zap.Error(err))
		response.ServerError(c, "生成 PDF 失败")
		return
	}
	defer os.Remove(htmlPath)

	h.logger.Info("HTML file created", zap.String("path", htmlPath))

	// 使用 Chrome 生成 PDF
	pdfPath, err := h.generatePDFWithChrome(htmlPath, filepath.Join(tmpDir, pdfPattern))
	if err != nil {
		h.logger.Error("Failed to generate PDF with Chrome", zap.Error(err))
		response.ServerError(c, "导出 PDF 失败")
		return
	}
	defer os.Remove(pdfPath)

	// 读取生成的 PDF
	pdfBytes, err := os.ReadFile(pdfPath)
	if err != nil {
		h.logger.Error("Failed to read PDF file", zap.Error(err))
		response.ServerError(c, "读取 PDF 失败")
		return
	}

	if len(pdfBytes) == 0 {
		h.logger.Error("Generated PDF is empty")
		response.ServerError(c, "导出 PDF 失败：生成内容为空")
		return
	}

	c.Header("Content-Type", "application/pdf")
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=%s-成长日记-%s.pdf",
		baby.Name, time.Now().Format("20060102")))
	c.Header("Content-Length", fmt.Sprintf("%d", len(pdfBytes)))

	c.Data(200, "application/pdf", pdfBytes)
}

// writeTempFile 创建临时文件并返回路径
func writeTempFile(fullPattern, content string) (string, error) {
	// 提取目录和文件名模式
	dir := filepath.Dir(fullPattern)
	pattern := filepath.Base(fullPattern)

	f, err := os.CreateTemp(dir, pattern)
	if err != nil {
		return "", err
	}
	defer f.Close()

	_, err = f.WriteString(content)
	if err != nil {
		os.Remove(f.Name())
		return "", err
	}

	return f.Name(), nil
}

// generatePDFWithChrome 使用 Chrome Headless 生成 PDF
func (h *ExportHandler) generatePDFWithChrome(htmlPath, pdfPattern string) (string, error) {
	// 创建输出文件
	dir := filepath.Dir(pdfPattern)
	pattern := filepath.Base(pdfPattern)

	f, err := os.CreateTemp(dir, pattern)
	if err != nil {
		return "", err
	}
	f.Close()
	outputPath := f.Name()

	// Chrome 路径
	chromePath := "/usr/bin/google-chrome"
	if _, err := os.Stat(chromePath); err != nil {
		// 尝试其他可能的路径
		paths := []string{
			"/usr/bin/chromium-browser",
			"/usr/bin/chromium",
			"/snap/bin/chromium",
		}
		for _, p := range paths {
			if _, err := os.Stat(p); err == nil {
				chromePath = p
				break
			}
		}
	}

	// 构建 Chrome 命令
	// 使用 file:// URL 并添加 --allow-file-access-from-files
	htmlURL := "file://" + htmlPath
	args := []string{
		"--headless",
		"--disable-gpu",
		"--no-sandbox",
		"--disable-setuid-sandbox",
		"--allow-file-access-from-files",
		"--print-to-pdf-no-header",
		"--print-to-pdf=" + outputPath,
		htmlURL,
	}

	cmd := exec.Command(chromePath, args...)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	h.logger.Error("DEBUG: Executing Chrome", zap.String("command", chromePath), zap.Strings("args", args), zap.String("htmlURL", htmlURL))

	if err := cmd.Run(); err != nil {
		h.logger.Error("Chrome command failed", zap.Error(err), zap.String("stderr", stderr.String()))
		return "", fmt.Errorf("chrome failed: %v, stderr: %s", err, stderr.String())
	}
	h.logger.Error("DEBUG: Chrome command succeeded")

	// 验证输出文件
	if _, err := os.Stat(outputPath); os.IsNotExist(err) {
		return "", fmt.Errorf("PDF file not created")
	}

	info, err := os.Stat(outputPath)
	if err != nil {
		return "", err
	}

	h.logger.Debug("PDF generated", zap.String("path", outputPath), zap.Int64("size", info.Size()))

	if info.Size() == 0 {
		return "", fmt.Errorf("PDF file is empty")
	}

	return outputPath, nil
}

// getImageFileName 从 URL 中提取图片文件名
func getImageFileName(url string) string {
	parts := strings.Split(url, "/")
	if len(parts) > 0 {
		return parts[len(parts)-1]
	}
	return url
}

// GetExportPreview 获取导出预览
func (h *ExportHandler) GetExportPreview(c *gin.Context) {
	_, exists := middleware.GetUserID(c)
	if !exists {
		response.Unauthorized(c, "未登录")
		return
	}

	babyIDStr := c.Query("baby_id")
	babyID, err := strconv.ParseUint(babyIDStr, 10, 32)
	if err != nil {
		response.ValidationError(c, "无效的 baby_id")
		return
	}

	baby, err := h.babyRepo.GetByID(uint(babyID))
	if err != nil {
		h.logger.Error("Failed to get baby", zap.Error(err))
		response.NotFound(c, "宝宝档案不存在")
		return
	}

	records, err := h.recordRepo.GetByBabyID(uint(babyID))
	if err != nil {
		h.logger.Error("Failed to get records", zap.Error(err))
		response.ServerError(c, "获取记录失败")
		return
	}

	imageCount := 0
	videoCount := 0
	for _, r := range records {
		var mediaItems []ExportMediaItem
		if r.Media != "" && r.Media != "[]" {
			json.Unmarshal([]byte(r.Media), &mediaItems)
		}
		for _, m := range mediaItems {
			if m.Type == "image" {
				imageCount++
			} else if m.Type == "video" {
				videoCount++
			}
		}
	}

	sort.Slice(records, func(i, j int) bool {
		return records[i].CreatedAt.Before(records[j].CreatedAt)
	})
	earliestDate := ""
	latestDate := ""
	if len(records) > 0 {
		earliestDate = toShanghaiTime(records[0].CreatedAt).Format("2006-01-02")
		latestDate = toShanghaiTime(records[len(records)-1].CreatedAt).Format("2006-01-02")
	}

	response.Success(c, gin.H{
		"baby_name":     baby.Name,
		"record_count":  len(records),
		"image_count":   imageCount,
		"video_count":   videoCount,
		"earliest_date": earliestDate,
		"latest_date":   latestDate,
	})
}
