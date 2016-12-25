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
	for x := img.Bounds().Min.X; x < img.Bounds().Dx(); x++ {
		for y := img.Bounds().Min.Y; y < img.Bounds().Dy(); y++ {
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

	//Interpolation Options (NearestNeighbor, Bilinear, Bicubic, MitchellNetravali, Lanczos2, Lanczos3)

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
