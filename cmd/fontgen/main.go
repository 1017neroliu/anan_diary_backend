package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/jung-kurt/gofpdf"
)

// 生成中文字体定义文件
// 使用方法：
// 1. 确保 fonts 目录下有 NotoSansCJK.ttf 字体文件
// 2. 运行此程序生成字体定义文件
// 3. 生成的 JSON 文件会保存在 fonts 目录下

func main() {
	// 获取脚本所在目录
	execPath, err := os.Executable()
	if err != nil {
		fmt.Printf("Error getting executable path: %v\n", err)
		return
	}

	// 尝试多个可能的 fonts 目录路径
	possiblePaths := []string{
		filepath.Join(filepath.Dir(execPath), "../../fonts"),
		filepath.Join(filepath.Dir(execPath), "../../../ad-backend/fonts"),
		"./fonts",
		"../fonts",
		"../../fonts",
	}

	var fontsDir string
	for _, path := range possiblePaths {
		if _, err := os.Stat(path); err == nil {
			fontsDir = path
			break
		}
	}

	if fontsDir == "" {
		// 创建 fonts 目录
		fontsDir = "./fonts"
		if err := os.MkdirAll(fontsDir, 0755); err != nil {
			fmt.Printf("Error creating fonts directory: %v\n", err)
			return
		}
	}

	fmt.Printf("Using fonts directory: %s\n", fontsDir)

	// 字体文件路径 - 优先使用 .ttf 文件
	fontPath := filepath.Join(fontsDir, "NotoSansCJK.ttf")
	if _, err := os.Stat(fontPath); os.IsNotExist(err) {
		fontPath = filepath.Join(fontsDir, "NotoSansCJK-Regular.ttc")
	}

	// 检查字体文件是否存在
	if _, err := os.Stat(fontPath); os.IsNotExist(err) {
		fmt.Printf("Font file not found: %s\n", fontPath)
		fmt.Println("Please copy NotoSansCJK.ttf to the fonts directory")
		return
	}

	// 使用 gofpdf 的 MakeFont 函数生成字体定义
	fontName := "NotoSansCJK"

	fmt.Println("Generating font definition for NotoSansCJK...")
	fmt.Println("This may take a few minutes...")

	// 对于中文字体，不需要 encoding 文件（使用空字符串）
	// MakeFont(fontFileStr, encodingFileStr, dstDirStr string, msgWriter io.Writer, embed bool)
	// 第二个参数是 encoding 文件路径，空字符串表示使用默认编码（Unicode）
	err = gofpdf.MakeFont(fontPath, "", fontsDir, os.Stdout, true)
	if err != nil {
		fmt.Printf("Error generating font definition: %v\n", err)
		fmt.Println("\nNote: For CJK fonts, you may need to use a Unicode encoding map")
		fmt.Println("Try copying a Unicode map file to the fonts directory")
		return
	}

	fmt.Printf("\nFont definition generated successfully!\n")
	fmt.Printf("Font file: %s/%s.json\n", fontsDir, fontName)
	fmt.Println("\nTo use this font in your PDF:")
	fmt.Println(`  pdf.AddFont("NotoSansCJK", "", "` + filepath.Join(fontsDir, fontName) + `.json")`)
	fmt.Println(`  pdf.SetFont("NotoSansCJK", "", 10)`)
}
