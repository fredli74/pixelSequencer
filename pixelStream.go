package main

/*
	#cgo CFLAGS: -Ipngquant/lib
	#cgo LDFLAGS: pngquant/lib/libimagequant.a
	#include "libimagequant.h"
*/
import "C"
import "unsafe"

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	"image/color/palette"
	"image/draw"
	"image/png"
	"io"
	"math"
	"os"
	"reflect"
	"strconv"
)

func panicOn(err error) {
	if err != nil {
		panic(err)
	}
}

func imageQuant(src *image.NRGBA, animation bool) (out *image.Paletted) {
	fmt.Printf("Quantizing image ... ")
	bounds := src.Bounds()
	w := bounds.Max.X - bounds.Min.X
	h := bounds.Max.Y - bounds.Min.Y

	// Setup imagequant attributes
	quantAttributes := C.liq_attr_create()
	if quantAttributes == nil {
		panic("Unable to initialize quantize attributes, imagequant library returned null")
	}
	defer C.liq_attr_destroy(quantAttributes)

	if animation {
		// Reserve one color for diff
		if err := C.liq_set_max_colors(quantAttributes, 255); err != C.LIQ_OK {
			panic(fmt.Sprintf("Unable to set quantize color count, imagequant library returned %v", err))
		}
	}

	// Set quantize speed to maximum quality
	if err := C.liq_set_speed(quantAttributes, 1); err != C.LIQ_OK {
		panic(fmt.Sprintf("Unable to set quantize speed, imagequant library returned %v", err))
	}

	// Prepare source image in memory
	quantImage := C.liq_image_create_rgba(quantAttributes, unsafe.Pointer(&src.Pix[0]), C.int(bounds.Max.X), C.int(bounds.Max.Y), 0)
	if quantImage == nil {
		panic("Unable to create quantized image, imagequant library returned null")
	}
	defer C.liq_image_destroy(quantImage)

	// Quantize image palette
	quantResult := C.liq_quantize_image(quantAttributes, quantImage)
	if quantResult == nil {
		panic("Unable to quantize image palette, imagequant library returned null")
	}
	defer C.liq_result_destroy(quantResult)

	// Create output image with a temporary default palette
	out = image.NewPaletted(image.Rect(0, 0, w, h), palette.Plan9)

	// Quantize the image data
	if quantErr := C.liq_write_remapped_image(quantResult, quantImage, unsafe.Pointer(&out.Pix[0] /*data[0]*/), C.size_t(w*h)); quantErr != C.LIQ_OK {
		panic(fmt.Sprintf("Unable to quantize image data, imagequant library returned %v", quantErr))
	}

	// Read the palette
	quantPalette := C.liq_get_palette(quantResult)
	if quantPalette == nil {
		panic("Unable to read quantized image palette, imagequant library returned null")
	}
	fmt.Printf("%v colors\n", quantPalette.count)

	// Set the output palette
	for x, c := range quantPalette.entries {
		out.Palette[x] = color.NRGBA{uint8(c.r), uint8(c.g), uint8(c.b), uint8(c.a)}
	}
	return
}

func quantize(in image.Image, animation bool) (out *image.Paletted) {
	bounds := in.Bounds()
	switch t := in.(type) {
	//	case *image.Paletted:
	//		return t
	case *image.NRGBA:
		return imageQuant(t, animation)
	default:
		// Convert to NRGBA from whatever it is
		nrgba := image.NewNRGBA(image.Rect(0, 0, bounds.Max.X, bounds.Max.Y))
		draw.Draw(nrgba, nrgba.Bounds(), in, image.Pt(0, 0), draw.Src)
		return imageQuant(nrgba, animation)
	}
}

func help() {
	fmt.Println("pixelStream v0.1 - (c)2016 by Fredrik Lidström")
	fmt.Println("")
	fmt.Println("pixelStream quantize <input.png> <output.png>")
	fmt.Println("   Quantize single image (png -> 8-bit)")
	fmt.Println("")
	fmt.Println("pixelStream unquantize <input.png> <output.png>")
	fmt.Println("   Unquantize single image (8-bit -> NRGBA png)")
	fmt.Println("")
	fmt.Println("pixelStream encode <input.png> <frame-count> <output.png>")
	fmt.Println("   Encode animation (vertical strip png -> 8-bit pixel stream):")
	fmt.Println("")
	fmt.Println("pixelStream decode <input.png> <frame-width> <frame-height> <output.png>")
	fmt.Println("   Decode animation image (8-bit pixel stream -> vertical strip NRGBA png")
	fmt.Println("")
	os.Exit(-1)
}

func main() {
	command := os.Args[1]
	switch command {
	case "quantize", "unquantize":
		if len(os.Args) < 3 {
			help()
		}
	case "encode":
		if len(os.Args) < 5 {
			help()
		}
	case "decode":
		if len(os.Args) < 6 {
			help()
		}
	default:
		help()
	}

	infile, err := os.Open(os.Args[2])
	panicOn(err)
	defer infile.Close()
	inputImage, _, err := image.Decode(infile)
	panicOn(err)

	bounds := inputImage.Bounds()
	if bounds.Min.X > 0 || bounds.Min.Y > 0 {
		panic("Images with minimum bounds > 0 is not supported")
	}
	inputW := bounds.Max.X
	inputH := bounds.Max.Y

	fmt.Printf("Input image (%s): %dx%d\n", reflect.TypeOf(inputImage), inputW, inputH)
	var outputImage image.Image

	switch command {
	case "quantize":
		outputImage = quantize(inputImage, false)
	case "unquantize":
		nrgba := image.NewNRGBA(image.Rect(0, 0, inputW, inputH))
		draw.Draw(nrgba, nrgba.Bounds(), inputImage, image.Pt(0, 0), draw.Src)
		outputImage = nrgba
	case "encode":
		frameW := inputW
		frameC, err := strconv.Atoi(os.Args[3])
		panicOn(err)

		if math.Remainder(float64(inputH), float64(frameC)) != 0 {
			panic(fmt.Sprintf("Image height %d is not even divisable by frame count %d", inputH, frameC))
		}
		frameH := inputH / frameC
		fmt.Printf("Number of frames: %d (%dx%d)\n", frameC, frameW, frameH)

		fmt.Println("Encoding pixel stream from vertical frame strip")
		strip := quantize(inputImage, false)
		//			strip := inputImage.(*image.NRGBA)

		// Re-arrange all frame pixels in a sequence
		streamImage := image.NewPaletted(image.Rect(0, 0, frameC, frameW*frameH), strip.Palette)
		//			streamImage := image.NewNRGBA(image.Rect(0, 0, frameC, frameW*frameH))
		{
			i := 0
			for y := 0; y < frameH; y++ {
				for x := 0; x < frameW; x++ {
					for f := 0; f < frameC; f++ {
						streamImage.Pix[i] = strip.Pix[f*frameH*frameW+y*frameW+x]
						//					streamImage.SetNRGBA(f, y*frameH+x, strip.NRGBAAt(x, f*frameH+y))
						i++
					}
				}
			}
		}
		outputImage = streamImage

		bestSize := int(^uint(0) >> 1)

		{
			fullSize := frameC * frameW * frameH
			testBuf := new(bytes.Buffer)

			for w := frameC * frameW; w <= frameC*frameW*frameH; w += frameC * frameW {
				if w > 300000 {
					w = 300000
				}
				h := int(math.Ceil(float64(fullSize) / float64(w)))
				wasted := w*h - fullSize
				/*if wasted == 0*/ {
					testImage := image.NewPaletted(image.Rect(0, 0, w, h), strip.Palette)
					//			testImage := image.NewNRGBA(image.Rect(0, 0, w, h))

					fmt.Printf("Output %d x %d  (%d wasted) ...  ", w, h, wasted)
					copy(testImage.Pix, streamImage.Pix)

					testBuf.Reset()
					err := png.Encode(testBuf, testImage)
					panicOn(err)
					fmt.Printf("%d bytes               \r", testBuf.Len())
					if testBuf.Len() < bestSize {
						fmt.Printf("\n")
						bestSize = testBuf.Len()

						out, err := os.Create(fmt.Sprintf("%d-%s", w, os.Args[len(os.Args)-1]))
						panicOn(err)
						io.Copy(out, testBuf)
						out.Close()
					}
				}
				if w >= 300000 {
					break
				}
			}
		}

	case "decode":
		frameW, err := strconv.Atoi(os.Args[3])
		panicOn(err)
		frameH, err := strconv.Atoi(os.Args[4])
		panicOn(err)
		frameC := (inputW * inputH) / (frameW * frameH)

		fmt.Printf("Number of frames: %d (%dx%d)\n", frameC, frameW, frameH)

		fmt.Println("Decoding pixel stream to vertical frame strip")
		strip := image.NewNRGBA(image.Rect(0, 0, frameW, frameH*frameC))
		{
			streamImage := inputImage.(*image.Paletted)
			i := 0
			for y := 0; y < frameH; y++ {
				for x := 0; x < frameW; x++ {
					for f := 0; f < frameC; f++ {
						c := streamImage.Palette[streamImage.Pix[i]]
						i++
						strip.Set(x, y+f*frameH, c)
					}
				}
			}
		}
		outputImage = strip

	default:
		help()
	}

	fmt.Printf("Output image (%s): %dx%d\n", reflect.TypeOf(outputImage), outputImage.Bounds().Max.X, outputImage.Bounds().Max.Y)
	out, err := os.Create(os.Args[len(os.Args)-1])
	panicOn(err)
	png.Encode(out, outputImage)
}
