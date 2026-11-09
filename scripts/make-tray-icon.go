// make-tray-icon 从应用图标生成彩色托盘图标：
// 高质量缩放到 32×32（16 points @2x Retina），保留原色。
// 用法: go run scripts/make-tray-icon.go <输入.png> <输出.png>
package main

import (
	"fmt"
	"image"
	"image/png"
	"os"

	xdraw "golang.org/x/image/draw"
)

func main() {
	if len(os.Args) != 3 {
		fmt.Fprintln(os.Stderr, "用法: go run scripts/make-tray-icon.go <输入.png> <输出.png>")
		os.Exit(1)
	}
	in, err := os.Open(os.Args[1])
	if err != nil {
		fmt.Fprintln(os.Stderr, "打开输入失败:", err)
		os.Exit(1)
	}
	defer in.Close()

	src, _, err := image.Decode(in)
	if err != nil {
		fmt.Fprintln(os.Stderr, "解码 PNG 失败:", err)
		os.Exit(1)
	}

	// 缩放到 32×32（16 points @2x Retina）
	dst := image.NewRGBA(image.Rect(0, 0, 32, 32))
	xdraw.CatmullRom.Scale(dst, dst.Bounds(), src, src.Bounds(), xdraw.Over, nil)

	out, err := os.Create(os.Args[2])
	if err != nil {
		fmt.Fprintln(os.Stderr, "创建输出失败:", err)
		os.Exit(1)
	}
	defer out.Close()

	enc := png.Encoder{CompressionLevel: png.BestCompression}
	if err := enc.Encode(out, dst); err != nil {
		fmt.Fprintln(os.Stderr, "编码 PNG 失败:", err)
		os.Exit(1)
	}
	fmt.Println("已生成:", os.Args[2])
}
