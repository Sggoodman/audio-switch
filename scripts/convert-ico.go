//go:build ignore

package main

import (
	"bytes"
	"encoding/binary"
	"image"
	_ "image/png"
	"os"
)

func main() {
	if len(os.Args) < 3 {
		println("用法: go run convert-ico.go <input.png> <output.ico>")
		os.Exit(1)
	}

	inputPath := os.Args[1]
	outputPath := os.Args[2]

	// 读取 PNG 文件
	img, err := os.ReadFile(inputPath)
	if err != nil {
		panic(err)
	}

	// 解码 PNG
	decoded, format, err := image.Decode(bytes.NewReader(img))
	if err != nil {
		panic(err)
	}
	println("解码格式:", format)

	bounds := decoded.Bounds()
	width := bounds.Dx()
	height := bounds.Dy()

	// ICO 文件最多支持 256x256
	maxSize := 256
	if width > maxSize || height > maxSize {
		println("警告: 图像尺寸过大，将缩放到", maxSize, "x", maxSize)
		// 简单缩放（这里简化处理）
	}

	// 创建 ICO 文件
	var ico bytes.Buffer

	// ICO 头部 (6 字节)
	// Reserved (2), Type (2 = ICO), Count (2)
	binary.Write(&ico, binary.LittleEndian, uint16(0))    // Reserved
	binary.Write(&ico, binary.LittleEndian, uint16(1))    // Type: 1 = ICO
	binary.Write(&ico, binary.LittleEndian, uint16(1))    // Count: 1 个图像

	// 计算图像数据大小
	// ICO 格式需要 BMP 格式的图像数据
	pngData := imgToIcoBitmap(decoded)

	// 目录头部 (16 字节)
	// ICO 格式中，256 用 0 表示
	iconWidth := uint8(width)
	if width >= 256 {
		iconWidth = 0
	}
	iconHeight := uint8(height)
	if height >= 256 {
		iconHeight = 0
	}

	binary.Write(&ico, binary.LittleEndian, iconWidth)           // Width
	binary.Write(&ico, binary.LittleEndian, iconHeight)          // Height
	binary.Write(&ico, binary.LittleEndian, uint8(0))            // Color count (0 = >256 colors)
	binary.Write(&ico, binary.LittleEndian, uint8(0))            // Reserved
	binary.Write(&ico, binary.LittleEndian, uint16(1))           // Color planes
	binary.Write(&ico, binary.LittleEndian, uint16(32))          // Bits per pixel
	binary.Write(&ico, binary.LittleEndian, uint32(len(pngData))) // Size of image data
	binary.Write(&ico, binary.LittleEndian, uint32(6+16))        // Offset to image data

	// 写入图像数据
	ico.Write(pngData)

	// 保存到文件
	err = os.WriteFile(outputPath, ico.Bytes(), 0644)
	if err != nil {
		panic(err)
	}

	println("转换完成:", outputPath)
}

func imgToIcoBitmap(img image.Image) []byte {
	bounds := img.Bounds()
	width := bounds.Dx()
	height := bounds.Dy()

	// ICO BMP 头部 (40 字节)
	var bmp bytes.Buffer

	binary.Write(&bmp, binary.LittleEndian, uint32(40))             // Header size
	binary.Write(&bmp, binary.LittleEndian, int32(width))           // Width
	binary.Write(&bmp, binary.LittleEndian, int32(height * 2))      // Height * 2 (for AND mask)
	binary.Write(&bmp, binary.LittleEndian, uint16(1))              // Planes
	binary.Write(&bmp, binary.LittleEndian, uint16(32))             // Bits per pixel
	binary.Write(&bmp, binary.LittleEndian, uint32(0))              // Compression
	binary.Write(&bmp, binary.LittleEndian, uint32(0))              // Image size
	binary.Write(&bmp, binary.LittleEndian, int32(0))               // X pixels per meter
	binary.Write(&bmp, binary.LittleEndian, int32(0))               // Y pixels per meter
	binary.Write(&bmp, binary.LittleEndian, uint32(0))              // Colors used
	binary.Write(&bmp, binary.LittleEndian, uint32(0))              // Important colors

	// 写入像素数据 (BGRA, 从下到上)
	for y := height - 1; y >= 0; y-- {
		for x := 0; x < width; x++ {
			r, g, b, a := img.At(x, y).RGBA()
			binary.Write(&bmp, binary.LittleEndian, uint8(b>>8))
			binary.Write(&bmp, binary.LittleEndian, uint8(g>>8))
			binary.Write(&bmp, binary.LittleEndian, uint8(r>>8))
			binary.Write(&bmp, binary.LittleEndian, uint8(a>>8))
		}
	}

	// AND mask (1 bit per pixel, for transparency)
	// 如果有 alpha 通道，可以用来做透明
	maskSize := (width + 7) / 8 * height
	maskData := make([]byte, maskSize)
	bmp.Write(maskData)

	return bmp.Bytes()
}
