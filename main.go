package main

import (
	"bytes"
	"fmt"
	"image"
	"io"
	"log"
	"net/http"

	"image/color"
	_ "image/gif"
	_ "image/jpeg"
	"image/png"
)

const MAX_PARSE_MEM int64 = 4 << 20 // 4MB
const INDEX_HTML = `
<html>
<head>
	<title>개지린다 진짜 ㅋㅋㅋ</title>
</head>
<body>
<a href="/merge">이미지 합치기</a>
<a href="/sep">이미지 분해하기</a>
</body>

`
const MERGE_HTML = `
<html>
<head>
	<title>개지린다 진짜 ㅋㅋㅋ</title>
</head>
<body>
<form enctype="multipart/form-data" action="/merge-post" method="post">
	<table>
		<tr>
			<td>흰 색 배경일 때 이미지:  </td>
			<td><input type="file" name="img_white" /></td>
		</tr>
		<tr>
			<td>검은 색 배경일 때 이미지:  </td>
			<td><input type="file" name="img_black" /></td>
		</tr>
		<tr>
			<td><input type="submit" value="합치기" /></td>
		</tr>
	</table>
</form>
</body>
</html>
`

func min(a, b int) int {
	if a > b {
		return b
	}

	return a
}

func main() {
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(INDEX_HTML))
	})

	http.HandleFunc("/merge", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(MERGE_HTML))
	})

	http.HandleFunc("/merge-post", func(w http.ResponseWriter, r *http.Request) {
		var bw, bb bytes.Buffer

		fw, _, err := r.FormFile("img_white")
		if err != nil {
			if err.Error() == http.ErrMissingFile.Error() {
				w.Write([]byte("흰 색일 때 이미지를 지정하지 않았습니다. 다시 시도해주세요."))
				return
			}
			log.Print("img_white FormFile errror:", err)
			w.WriteHeader(http.StatusForbidden)
			return
		}
		defer fw.Close()
		io.Copy(&bw, fw)

		fb, _, err := r.FormFile("img_black")
		if err != nil {
			if err.Error() == http.ErrMissingFile.Error() {
				w.Write([]byte("검은 색일 때 이미지를 지정하지 않았습니다. 다시 시도해주세요."))
				return
			}
			log.Print("img_black FormFile errror:", err)
			w.WriteHeader(http.StatusForbidden)
			return
		}
		defer fb.Close()
		io.Copy(&bb, fb)

		img_w, _, err := image.Decode(&bw)
		if err != nil {
			log.Print("img_w parse error:", err)
			w.Write([]byte("이미지 처리 중 오류가 발생했습니다. 다시 시도해 보세요."))
			return
		}
		img_b, _, err := image.Decode(&bb)
		if err != nil {
			log.Print("img_b parse error:", err)
			w.Write([]byte("이미지 처리 중 오류가 발생했습니다. 다시 시도해 보세요."))
			return
		}

		wBounds := img_w.Bounds()
		bBounds := img_b.Bounds()

		maxDx := min(wBounds.Dx(), bBounds.Dx())
		maxDy := min(wBounds.Dy(), bBounds.Dy())

		out := image.NewRGBA(image.Rectangle{image.Point{0, 0}, image.Point{maxDx, maxDy}})
		outBounds := out.Bounds()

		for y := 0; y < maxDy; y++ {
			for x := 0; x < maxDx; x++ {
				cw := img_w.At(wBounds.Min.X+x, wBounds.Min.Y+y)
				cb := img_b.At(bBounds.Min.X+x, bBounds.Min.Y+y)

				kw := uint8(color.Gray16Model.Convert(cw).(color.Gray16).Y >> 8)
				kw = uint8((0xff + int(kw)) / 2)
				kb := uint8(color.Gray16Model.Convert(cb).(color.Gray16).Y >> 8)
				kb /= 2

				var a, k uint8
				if kb == 0 {
					a = 0xff - kw
					k = 0
				} else {
					a = uint8(0xff + kb - kw)
					k = uint8(0xff * float32(kb) / float32(a))
				}
				if kb > a {
					panic(fmt.Sprintln("a < kb ?? : a", a, "kb", kb))
				}
				out.Set(outBounds.Min.X+x, outBounds.Min.Y+y, color.NRGBA{k, k, k, a})
				k = k - k
			}
		}
		// w.Header().Set("Content-Disposition", "attachment; filename=composite.png")
		w.Header().Set("Content-Type", "image/png")

		if err := png.Encode(w, out); err != nil {
			log.Print("out image encode/send error:", err)
		}

	})

	http.ListenAndServe(":8080", nil)
}
