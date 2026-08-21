// Package thumbnail 提供图片缩放功能。
package thumbnail

import (
	"image"
	_ "image/gif" // 注册 GIF 解码器
	"image/jpeg"
	_ "image/png" // 注册 PNG 解码器
	"io"
	"os"
	"path/filepath"
	"strings"
)

// maxDim 是缩略图长边的像素上限。
const maxDim = 128

// Image 把 src 缩放成缩略图大小，保持原始宽高比。
func Image(src image.Image) image.Image {
	xs := src.Bounds().Dx()
	ys := src.Bounds().Dy()
	if xs == 0 || ys == 0 {
		return src
	}

	width, height := maxDim, maxDim
	if aspect := float64(xs) / float64(ys); aspect < 1.0 {
		width = int(maxDim * aspect) // 竖图
	} else {
		height = int(maxDim / aspect) // 横图
	}
	if width < 1 {
		width = 1
	}
	if height < 1 {
		height = 1
	}

	xscale := float64(xs) / float64(width)
	yscale := float64(ys) / float64(height)

	dst := image.NewRGBA(image.Rect(0, 0, width, height))
	// 最近邻采样，算法简单但足以说明问题。
	for x := 0; x < width; x++ {
		for y := 0; y < height; y++ {
			srcx := src.Bounds().Min.X + int(float64(x)*xscale)
			srcy := src.Bounds().Min.Y + int(float64(y)*yscale)
			dst.Set(x, y, src.At(srcx, srcy))
		}
	}
	return dst
}

// ImageStream 从 r 读取一张图片，把它的缩略图写入 w。
// 输入可以是 GIF、PNG 或 JPEG，输出始终是 JPEG。
func ImageStream(w io.Writer, r io.Reader) error {
	src, _, err := image.Decode(r)
	if err != nil {
		return err
	}
	return jpeg.Encode(w, Image(src), nil)
}

// ImageFile reads an image from infile and writes
// a thumbnail-size version of it in the same directory.
// It returns the generated file name, e.g., "foo.thumb.jpg".
func ImageFile(infile string) (string, error) {
	f, err := os.Open(infile)
	if err != nil {
		return "", err
	}
	defer f.Close()

	src, _, err := image.Decode(f)
	if err != nil {
		return "", err
	}

	ext := filepath.Ext(infile) // 例如 ".jpg"、".JPEG"
	outfile := strings.TrimSuffix(infile, ext) + ".thumb" + ext

	out, err := os.Create(outfile)
	if err != nil {
		return "", err
	}
	if err := jpeg.Encode(out, Image(src), nil); err != nil {
		out.Close()
		return "", err
	}
	// 关闭写入的文件时也要检查错误，否则可能丢掉缓冲区里的数据。
	if err := out.Close(); err != nil {
		return "", err
	}
	return outfile, nil
}

func makeThumbnails3(filenames []string) {
	ch := make(chan struct{})
	for _, filename := range filenames {
		go func(f string) {
			outfile, err := ImageFile(f)
			if err != nil {
				ch <- struct{}{}
				return
			}
			_ = outfile
			ch <- struct{}{}
		}(filename)
	}

	for range filenames {
		<-ch
	}
}

func makeThumbnails4(filenames []string) {
	errors := make(chan error)
	for _, filename := range filenames {
		go func(f string) {
			_, err := ImageFile(f)
			errors <- err
		}(filename)
	}
	for range filenames {
		<-errors
	}
}

func makeThumbnails5(filenames []string) (thumbnails []string, err error) {
	type item struct {
		thumbfile string
		err       error
	}

	ch := make(chan item, len(filenames))
	for _, filename := range filenames {
		go func(f string) {
			var it item
			it.thumbfile, it.err = ImageFile(f)
			ch <- it
		}(filename)
	}
	for range filenames {
		it := <-ch
		if it.err != nil {
			return nil, it.err
		}
		thumbnails = append(thumbnails, it.thumbfile)
	}
	return thumbnails, nil
}
