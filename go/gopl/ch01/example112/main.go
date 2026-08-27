package main

import (
	"fmt"
	"image"
	"image/color"
	"image/gif"
	"io"
	"log"
	"math"
	"math/rand"
	"net/http"
	"strconv"
)

var palette = []color.Color{
	color.White,
	color.RGBA{0, 100, 0, 1},
	color.RGBA{72, 61, 139, 1},
	color.RGBA{220, 20, 60, 1},
	color.RGBA{0, 255, 255, 1},
	color.RGBA{0, 0, 139, 1},
	color.RGBA{148, 0, 211, 1},
}

func main() {
	http.HandleFunc("/", lissajousHandler)
	log.Fatal(http.ListenAndServe("127.0.0.1:8080", nil))
}

func lissajousHandler(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	cyclesParams := r.Form.Get("cycles")
	var cycles int
	if cyclesParams != "" {
		var err error
		cycles, err = strconv.Atoi(cyclesParams)
		if err != nil {
			fmt.Fprint(w, "please input number cycles", err)
			return
		}
	}
	if cycles <= 0 {
		cycles = 5
	}
	lissajous(w, cycles)
}

func lissajous(out io.Writer, cycles int) {
	const (
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

		paletteIndex := i%(len(palette)-1) + 1
		for t := 0.0; t < float64(cycles)*2*math.Pi; t += res {
			x := math.Sin(t)
			y := math.Sin(t*freq + phase)
			img.SetColorIndex(size+int(x*size+0.5), size+int(y*size+0.5), uint8(paletteIndex))
		}
		phase += 0.1
		anim.Delay = append(anim.Delay, delay) // 延迟80ms
		anim.Image = append(anim.Image, img)   // 添加一帧
	}
	gif.EncodeAll(out, &anim)
}
