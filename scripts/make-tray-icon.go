// make-tray-icon 从彩色图标生成 macOS 菜单栏 template 托盘图标：
// 保留 alpha 形状，RGB 置为纯黑（IconIsTemplate 模式下系统按菜单栏深浅自动着色）。
// 用法: go run scripts/make-tray-icon.go <输入.png> <输出.png>
package main

import (
	"fmt"
	"image"
	"image/color"
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

	// template 化：纯黑 + 原 alpha（低于 40% 的半透明像素直接透明，避免边缘灰晕）
	for y := 0; y < 32; y++ {
		for x := 0; x < 32; x++ {
			_, _, _, a := dst.At(x, y).RGBA()
			alpha := uint8(a >> 8)
			if alpha < 100 {
				alpha = 0
			}
			dst.SetRGBA(x, y, color.RGBA{R: 0, G: 0, B: 0, A: alpha})
		}
	}

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
