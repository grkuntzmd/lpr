package main

import (
	"flag"
	"fmt"
	"image"
	"image/color"
	"image/jpeg"
	"math"
	"os"

	"golang.org/x/image/tiff"
	_ "golang.org/x/image/tiff"
	"gonum.org/v1/gonum/mat"
)

var input, output string

func init() {
	flag.StringVar(&input, "input", "", "input file")
	flag.StringVar(&output, "output", "", "output file")
}

func main() {
	flag.Parse()

	if len(input) == 0 {
		fmt.Fprintf(os.Stderr, "missing input file name\n")
		flag.Usage()
		os.Exit(1)
	}

	if len(output) == 0 {
		fmt.Fprintf(os.Stderr, "missing output file name\n")
		flag.Usage()
		os.Exit(1)
	}

	f, err := os.Open(input)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err)
		os.Exit(255)
	}
	defer f.Close()

	inImg, imageType, err := image.Decode(f)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err)
		os.Exit(255)
	}

	bounds := inImg.Bounds()

	coordsData := make([]float64, 0, bounds.Max.X*bounds.Max.Y*6)

	for y := 0; y < bounds.Max.Y; y++ {
		for x := 0; x < bounds.Max.X; x++ {
			coordsData = append(coordsData, 1, float64(x), float64(y), float64(x*x), float64(y*y), float64(x*y))
		}
	}

	coords := mat.NewDense(bounds.Max.X*bounds.Max.Y, 6, coordsData)

	red := getSolution(inImg, coords, func(c color.Color) float64 {
		r, _, _, _ := c.RGBA()
		return float64(r)
	})

	green := getSolution(inImg, coords, func(c color.Color) float64 {
		_, g, _, _ := c.RGBA()
		return float64(g)
	})

	blue := getSolution(inImg, coords, func(c color.Color) float64 {
		_, _, b, _ := c.RGBA()
		return float64(b)
	})

	outImg := image.NewNRGBA64(bounds)
	for y := 0; y < bounds.Max.Y; y++ {
		for x := 0; x < bounds.Max.X; x++ {
			outImg.Set(x, y, color.NRGBA64{pixel(red, x, y), pixel(green, x, y), pixel(blue, x, y), 0xffff})
		}
	}

	out, err := os.OpenFile(output, os.O_CREATE|os.O_WRONLY, 0755)
	if err != nil {
		fmt.Fprintf(os.Stderr, "cannot open output file %s for writing: %s\n", output, err)
		os.Exit(2)
	}

	switch imageType {
	case "jpeg":
		err = jpeg.Encode(out, outImg, nil)
		if err != nil {
			fmt.Fprintf(os.Stderr, "cannot write JPEG image to file %s: %s\n", output, err)
			os.Exit(2)
		}

	case "tiff":
		err = tiff.Encode(out, outImg, &tiff.Options{Compression: tiff.Deflate, Predictor: true})
		if err != nil {
			fmt.Fprintf(os.Stderr, "cannot write TIFF image to file %s: %s\n", output, err)
			os.Exit(2)
		}

	default:
		fmt.Fprintf(os.Stderr, "unknown image format\n")
	}

}

func getSolution(i image.Image, coords *mat.Dense, cf func(color.Color) float64) *mat.Dense {
	values := getValues(i, cf)
	solution := mat.NewDense(6, 1, nil)
	solution.Solve(coords, values)
	return solution
}

func getValues(i image.Image, cf func(color.Color) float64) *mat.Dense {
	bounds := i.Bounds()
	values := make([]float64, 0, bounds.Max.X*bounds.Max.Y)
	for y := 0; y < bounds.Max.Y; y++ {
		for x := 0; x < bounds.Max.X; x++ {
			values = append(values, cf(i.At(x, y)))
		}
	}

	return mat.NewDense(bounds.Max.X*bounds.Max.Y, 1, values)
}

func pixel(d *mat.Dense, x, y int) uint16 {
	return uint16(math.Max(d.At(0, 0)+
		d.At(1, 0)*float64(x)+
		d.At(2, 0)*float64(y)+
		d.At(3, 0)*float64(x*x)+
		d.At(4, 0)*float64(y*y)+
		d.At(5, 0)*float64(x*y), 0))
}
