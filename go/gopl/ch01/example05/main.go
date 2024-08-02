package main

import (
	"image"
	"image/color"
	"image/gif"
	"io"
	"math"
	"math/rand"
	"os"
)

var palette = []color.Color{color.White, color.RGBA{0, 127, 0, 128}}

const (
	whiteIndex = 0 //palette中的第一个元素
	blackIndex = 1 //palette中的第二个元素
)

func main() {
	lissajous(os.Stdout)
}

func lissajous(out io.Writer) {
	const (
		cycles = 5    //完整x振荡器转数
		res    = 0.01 //角分辨率
		size   = 100  //图像画布封面[-size..+size]
		nframe = 64   //动画帧数
		delay  = 8    //以10ms为单位的帧间延迟
	)

	freq := rand.Float64() * 3.0 //
	anim := gif.GIF{LoopCount: nframe}
	phase := 0.0
	for i := 0; i < nframe; i++ {
		rect := image.Rect(0, 0, 2*size+1, 2*size+1) // 画布大小
		img := image.NewPaletted(rect, palette)      // 所有像素点都会被设置为其0值，也就是第一个palette的值
		for t := 0.0; t < cycles*2*math.Pi; t += res {
			x := math.Sin(t)
			y := math.Sin(t*freq + phase)
			img.SetColorIndex(size+int(x*size+0.5), size+int(y*size+0.5), blackIndex)
		}
		phase += 0.1
		anim.Delay = append(anim.Delay, delay) // 延迟80ms
		anim.Image = append(anim.Image, img)   // 添加一帧
	}
	gif.EncodeAll(out, &anim)
}
