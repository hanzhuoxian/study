// surface 根据 URL 参数 expr 给出的表达式，
// 计算并渲染出该三维曲面函数的 SVG 图像。
//
// 用法：
//
//	go run ./ch07/surface
//	http://localhost:8000/plot?expr=sin(-x)*pow(1.5,-r)
package main

import (
	"fmt"
	"log"
	"math"
	"net/http"

	"github.com/hanzhuoxian/study/go/gopl/ch07/eval"
)

const (
	width, height = 600, 320            // 画布大小（像素）
	cells         = 100                 // 网格单元数
	xyrange       = 30.0                // 坐标轴范围（-xyrange...+xyrange）
	xyscale       = width / 2 / xyrange // 每个 x/y 单位对应的像素数
	zscale        = height * 0.4        // 每个 z 单位对应的像素数
	angle         = math.Pi / 6         // x、y 轴的倾斜角（=30°）
)

var sin30, cos30 = math.Sin(angle), math.Cos(angle) // sin(30°)、cos(30°)

func main() {
	http.HandleFunc("/plot", plot)
	log.Println("listening on http://localhost:8000/plot?expr=sin(-x)*pow(1.5,-r)")
	log.Fatal(http.ListenAndServe("localhost:8000", nil))
}

func parseAndCheck(s string) (eval.Expr, error) {
	if s == "" {
		return nil, fmt.Errorf("empty expression")
	}
	expr, err := eval.Parse(s)
	if err != nil {
		return nil, err
	}
	vars := make(map[eval.Var]bool)
	if err := expr.Check(vars); err != nil {
		return nil, err
	}
	for v := range vars {
		if v != "x" && v != "y" && v != "r" {
			return nil, fmt.Errorf("undefined variable: %s", v)
		}
	}
	return expr, nil
}

func plot(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	expr, err := parseAndCheck(r.Form.Get("expr"))
	if err != nil {
		http.Error(w, "bad expr: "+err.Error(), http.StatusBadRequest)
		return
	}
	w.Header().Set("Content-Type", "image/svg+xml")
	surface(w, func(x, y float64) float64 {
		r := math.Hypot(x, y) // 到原点 (0,0) 的距离
		return expr.Eval(eval.Env{"x": x, "y": y, "r": r})
	})
}

// surface 把曲面函数 f 渲染成 SVG 写入 w。
func surface(w http.ResponseWriter, f func(x, y float64) float64) {
	fmt.Fprintf(w, "<svg xmlns='http://www.w3.org/2000/svg' "+
		"style='stroke: grey; fill: white; stroke-width: 0.7' "+
		"width='%d' height='%d'>\n", width, height)
	for i := 0; i < cells; i++ {
		for j := 0; j < cells; j++ {
			ax, ay, ok := corner(i+1, j, f)
			if !ok {
				continue
			}
			bx, by, ok := corner(i, j, f)
			if !ok {
				continue
			}
			cx, cy, ok := corner(i, j+1, f)
			if !ok {
				continue
			}
			dx, dy, ok := corner(i+1, j+1, f)
			if !ok {
				continue
			}
			fmt.Fprintf(w, "<polygon points='%g,%g,%g,%g,%g,%g,%g,%g'/>\n",
				ax, ay, bx, by, cx, cy, dx, dy)
		}
	}
	fmt.Fprintln(w, "</svg>")
}

// corner 返回网格单元 (i, j) 一个角在 SVG 画布上的投影坐标。
// 若该点的 z 值不是有限数（NaN 或 ±Inf），返回的第三个值为 false，
// 调用方应跳过包含该角的多边形。
func corner(i, j int, f func(x, y float64) float64) (float64, float64, bool) {
	// 求网格单元 (i, j) 对应的点 (x, y)
	x := xyrange * (float64(i)/cells - 0.5)
	y := xyrange * (float64(j)/cells - 0.5)

	// 计算曲面高度 z
	z := f(x, y)
	if math.IsNaN(z) || math.IsInf(z, 0) {
		return 0, 0, false
	}

	// 把 (x, y, z) 等角投影到二维 SVG 画布 (sx, sy) 上
	sx := width/2 + (x-y)*cos30*xyscale
	sy := height/2 + (x+y)*sin30*xyscale - z*zscale
	return sx, sy, true
}
