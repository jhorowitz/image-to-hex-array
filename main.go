package main

import (
	// Imported to allow image.Decode to detect png
	_ "image/png"
	// Imported to allow image.Decode to detect jpeg
	_ "image/jpeg"

	"flag"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"io/ioutil"
	"os"
	"strings"

	"github.com/Sirupsen/logrus"
	"github.com/TV4/env"
	"github.com/nfnt/resize"
)

var debugMode = env.Bool("DEBUG_MODE", false)

func init() {
	if debugMode {
		logrus.SetLevel(logrus.DebugLevel)
	}
}

var (
	width, height    int
	imagePath        string
	outputFilePath   string
	calculateOpacity bool
)

func init() {
	flag.IntVar(&width, "width", 300, "Output width(number of image slices)")
	flag.IntVar(&height, "height", 72, "Output Height(number of leds on Poi)")

	flag.StringVar(&imagePath, "image", "", "The image file to convert")
	flag.StringVar(&outputFilePath, "output", "./output.txt", "Output Hex")

	flag.BoolVar(&calculateOpacity, "calculate-opacity", false, "Calculate the opacity into the hex value")

	flag.Parse()

	if len(imagePath) == 0 {
		logrus.Error("An image path must be set. Try --help for more information")
		os.Exit(1)
	}

	logrus.WithFields(logrus.Fields{
		"width":          width,
		"height":         height,
		"imagePath":      imagePath,
		"outputFilePath": outputFilePath,
		"includeOpacity": calculateOpacity,
	}).Debug("Flags Set")
}

func main() {
	img := getImage(imagePath)
	img = resizeImage(uint(width), uint(height), img)

	hex := toHex(img, false)
	saveHex(hex, outputFilePath)
}

func outputToImage() {
	hexInput, err := os.Open("output.txt")
	if err != nil {
		panic(err)
	}

	b, err := ioutil.ReadAll(hexInput)
	if err != nil {
		panic(err)
	}

	hexArr := hexOutputToHexArr(string(b))
	imgFromHex := hexToImage(300, 72, hexArr)

	handle, err := os.Create("from-hex.png")
	if err != nil {
		panic(err)
	}

	err = png.Encode(handle, imgFromHex)
	if err != nil {
		panic(err)
	}

	return
	//saveHex(hex, outputFilePath)
}

func getImage(path string) image.Image {
	handle, err := os.Open(path)
	if err != nil {
		panic(err)
	}

	img, format, err := image.Decode(handle)
	if err != nil {
		panic(fmt.Sprintf("Could not decode image: %s", err))
	}
	defer handle.Close()
	logrus.WithField("Format", format).Debug("Image format detected")

	return img
}

func saveHex(rgbHex []string, path string) {
	handle, err := os.Create(path)
	if err != nil {
		panic(err)
	}

	_, err = handle.WriteString("\n")
	if err != nil {
		panic(err)
	}

	outputFormat := toOutputFormat(rgbHex)
	_, err = handle.WriteString(outputFormat)
	if err != nil {
		panic(err)
	}
}

func toOutputFormat(input []string) string {
	const start = "const unsigned int array1[] = {"
	const end = ", }; //end of array "

	var hexi = strings.Join(input, ", ")

	return start + hexi + end
}

func toHex(img image.Image, withOpacity bool) []string {
	var result []string
	// x == width, y == height
	for y := img.Bounds().Min.Y; y < img.Bounds().Dy(); y++ {
		for x := img.Bounds().Min.X; x < img.Bounds().Dx(); x++ {
			var hexVal = colorToHexValue(img.At(x, y), withOpacity)
			result = append(result, hexVal)
		}
	}

	return result
}

func colorToHexValue(c color.Color, withOpacity bool) string {
	r, g, b, a := c.RGBA()

	if withOpacity {
		r = ((1 - a) * r) + (a * r)
		g = ((1 - a) * g) + (a * g)
		b = ((1 - a) * b) + (a * b)
	}

	return fmt.Sprintf("0x%02x%02x%02x", uint8(r), uint8(g), uint8(b))
}

func resizeImage(width, height uint, img image.Image) image.Image {
	if debugMode {
		f, err := os.Create("before_resize.png")
		if err != nil {
			panic(err)
		}
		defer f.Close()
		if err = png.Encode(f, img); err != nil {
			panic(err)
		}
	}

	// Interpolation Options (NearestNeighbor, Bilinear, Bicubic, MitchellNetravali, Lanczos2, Lanczos3)

	img = resize.Resize(width, height, img, resize.Lanczos3)

	if debugMode {
		f, err := os.Create("after_resize.png")
		if err != nil {
			panic(err)
		}
		defer f.Close()
		if err = png.Encode(f, img); err != nil {
			panic(err)
		}
	}

	return img
}

func hexOutputToHexArr(output string) []string {
	output = strings.Replace(output, ", }", "", -1)
	output = strings.Replace(output, ";", "", -1)
	output = strings.Replace(output, ";", "", -1)
	output = strings.Replace(output, "//end of array", "", -1)
	output = strings.Replace(output, "const unsigned int array1[] =", "", -1)
	output = strings.Replace(output, "{", "", -1)
	output = strings.Replace(output, "}", "", -1)
	output = strings.Replace(output, " ", "", -1)
	output = strings.Replace(output, "\t", "", -1)
	output = strings.Replace(output, "\n", "", -1)
	output = strings.Replace(output, "\v", "", -1)
	output = strings.Replace(output, "\f", "", -1)
	output = strings.Replace(output, "\r", "", -1)

	return strings.Split(output, ",")
}

func hexToImage(width, height int, hexArr []string) image.Image {
	var c [][]color.Color

	for i := 0; i < width; i++ {
		c = append(c, nil)
		for j := 0; j < height; j++ {
			c[i] = append(c[i], nil)
		}
	}

	var idx int
	for j := 0; j < height; j++ {
		for i := 0; i < width; i++ {
			c[i][j] = hexToColor(hexArr[idx])
			idx++
		}
	}

	return HexImage(c)
}

type HexImage [][]color.Color

func (h HexImage) ColorModel() color.Model {
	var f = func(c color.Color) color.Color {
		r, g, b, a := c.RGBA()
		result := color.RGBA{
			R: uint8(r),
			G: uint8(g),
			B: uint8(b),
			A: uint8(a),
		}
		fmt.Println("RESULT", result)
		return result
	}
	return color.ModelFunc(f)
}

func (h HexImage) Bounds() image.Rectangle {
	return image.Rectangle{
		Min: image.Point{0, 0},
		Max: image.Point{X: len(h), Y: len(h[0])},
	}
}

func (h HexImage) At(x, y int) color.Color {
	return h[x][y]
}

func hexToColor(hex string) color.Color {
	var r, g, b uint8

	_, err := fmt.Fscanf(strings.NewReader(hex), "0x%02x%02x%02x", &r, &g, &b)
	if err != nil {
		panic(err)
	}

	return color.RGBA{
		R: uint8(r),
		G: uint8(g),
		B: uint8(b),
		A: 255,
	}
}
