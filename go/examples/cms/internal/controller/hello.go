package controller

import (
	"context"

	"github.com/gogf/gf/v2/frame/g"

	v1 "cms/api/v1"
)

var (
	Hello = cHello{}
)

type cHello struct{}

func (c *cHello) Hello(ctx context.Context, req *v1.HelloReq) (res *v1.HelloRes, err error) {
	g.RequestFromCtx(ctx).Response.Writeln("Hello World!")
	return
}
func generate(numRows int) [][]int {
	triangle := make([][]int, numRows)
	for row := 1; row <= numRows; row++ {
		rIndex := row - 1
		triangle[rIndex] = make([]int, row)
		for column := 0; column < row; column++ {
			if column == 0 || column == row-1 {
				triangle[rIndex][column] = 1
				continue
			}
			triangle[rIndex][column] = triangle[rIndex-1][column-1] + triangle[rIndex-1][column]
		}
	}
	return triangle
}
