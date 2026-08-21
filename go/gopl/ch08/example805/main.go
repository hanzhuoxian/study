// Mandelbrot emits a PNG image of the Mandelbrot fractal.
package main

// **练习 8.5：** 使用一个已有的CPU绑定的顺序程序，比如在3.3节中我们写的Mandelbrot程序或者3.2节中的3-D surface计算程序，
// 并将他们的主循环改为并发形式，使用channel来进行通信。在多核计算机上这个程序得到了多少速度上的改进？
// 使用多少个goroutine是最合适的呢？
import (
	"bufio"
	"flag"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"math/cmplx"
	"os"
	"runtime"
	"sync"
	"time"
)

const (
	xmin, ymin, xmax, ymax = -2, -2, +2, +2
	width, height          = 1024, 1024
)

var (
	workers = flag.Int("workers", runtime.NumCPU(), "并发渲染的 goroutine 数量，0 表示用顺序版本")
	bench   = flag.Bool("bench", false, "跑一遍不同 goroutine 数量的基准测试，不输出图片")
	outfile = flag.String("o", "", "输出的 PNG 文件名，留空则写到标准输出")
)

func main() {
	flag.Parse()

	if *bench {
		runBench()
		return
	}

	var img *image.RGBA
	if *workers <= 0 {
		img = renderSeq()
	} else {
		img = renderPar(*workers)
	}

	if err := encode(img, *outfile); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

// renderSeq 是原始的顺序实现，留作性能基准。
func renderSeq() *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	for py := range height {
		renderRow(img, py)
	}
	return img
}

// renderPar 用 workers 个 goroutine 并发渲染，行号通过 channel 分发。
//
// 每个 goroutine 只写自己那一行的像素，各行在 img.Pix 里的区间互不重叠，
// 所以共享 img 不需要加锁，也不存在数据竞争（可以用 -race 验证）。
//
// 这里没有把 1024 行静态切成 workers 块，而是让所有 goroutine 从同一个
// channel 抢任务：Mandelbrot 各行的计算量差别很大（集合内部的点要迭代满
// 200 次，外面的点很快就发散），静态切分会让某些 goroutine 早早空转。
func renderPar(workers int) *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, width, height))

	rows := make(chan int, height)
	for py := range height {
		rows <- py
	}
	close(rows)

	var wg sync.WaitGroup
	for range workers {
		wg.Go(func() {
			for py := range rows {
				renderRow(img, py)
			}
		})
	}
	wg.Wait()

	return img
}

// renderRow 计算第 py 行的所有像素。
func renderRow(img *image.RGBA, py int) {
	y := float64(py)/height*(ymax-ymin) + ymin
	for px := range width {
		x := float64(px)/width*(xmax-xmin) + xmin
		z := complex(x, y)
		// Image point (px, py) represents complex value z.
		img.Set(px, py, mandelbrot(z))
	}
}

func mandelbrot(z complex128) color.Color {
	const iterations = 200
	const contrast = 15

	var v complex128
	for n := uint8(0); n < iterations; n++ {
		v = v*v + z
		if cmplx.Abs(v) > 2 {
			return color.Gray{255 - contrast*n}
		}
	}
	return color.Black
}

func encode(img *image.RGBA, name string) error {
	if name == "" {
		w := bufio.NewWriter(os.Stdout)
		if err := png.Encode(w, img); err != nil {
			return err
		}
		return w.Flush()
	}

	f, err := os.Create(name)
	if err != nil {
		return err
	}
	if err := png.Encode(f, img); err != nil {
		f.Close()
		return err
	}
	return f.Close()
}

// runBench 依次测量顺序版本和不同 goroutine 数量的并发版本，
// 每种配置跑 3 轮取最快的一次，并顺手校验并发结果和顺序结果逐字节一致。
func runBench() {
	const rounds = 3

	fmt.Printf("GOMAXPROCS=%d NumCPU=%d 图片 %dx%d\n\n",
		runtime.GOMAXPROCS(0), runtime.NumCPU(), width, height)

	want := renderSeq()
	base := best(rounds, func() *image.RGBA { return renderSeq() })
	fmt.Printf("%-12s %10s %8s\n", "配置", "耗时", "加速比")
	fmt.Printf("%-12s %10s %8s\n", "顺序", base.Round(time.Millisecond), "1.00x")

	for _, n := range workerCounts() {
		var got *image.RGBA
		d := best(rounds, func() *image.RGBA {
			got = renderPar(n)
			return got
		})
		label := fmt.Sprintf("%d goroutine", n)
		if !equal(got, want) {
			label += " (结果不一致!)"
		}
		fmt.Printf("%-12s %10s %7.2fx\n", label, d.Round(time.Millisecond),
			float64(base)/float64(d))
	}
}

// workerCounts 生成待测的 goroutine 数量：1、2、4…直到 4 倍核数。
func workerCounts() []int {
	max := 4 * runtime.NumCPU()
	var ns []int
	for n := 1; n <= max; n *= 2 {
		ns = append(ns, n)
	}
	return ns
}

func best(rounds int, render func() *image.RGBA) time.Duration {
	d := time.Duration(1 << 62)
	for range rounds {
		start := time.Now()
		render()
		if e := time.Since(start); e < d {
			d = e
		}
	}
	return d
}

func equal(a, b *image.RGBA) bool {
	if a.Rect != b.Rect || len(a.Pix) != len(b.Pix) {
		return false
	}
	for i := range a.Pix {
		if a.Pix[i] != b.Pix[i] {
			return false
		}
	}
	return true
}
