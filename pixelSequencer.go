package main

import "C"

import (
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/png"
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

func clamp(i int32) int32 {
	if i < 0 {
		return 0
	}
	if i > 0xffff {
		return 0xffff
	}
	return i
}

func floydSteinberg(src *image.NRGBA64) (dst *image.NRGBA) {
	r := src.Bounds()
	nrgba := image.NewNRGBA(r)

	var quantErrorCurr, quantErrorNext [][4]int32
	quantErrorCurr = make([][4]int32, r.Dx()+2)
	quantErrorNext = make([][4]int32, r.Dx()+2)

	var out color.NRGBA
	for y := 0; y != r.Dy(); y++ {
		for x := 0; x != r.Dx(); x++ {
			// source color
			sc := src.At(x, y).(color.NRGBA64)
			// target color
			tr, tg, tb, ta := int32(sc.R), int32(sc.G), int32(sc.B), int32(sc.A)
			tr = clamp(tr + quantErrorCurr[x+1][0]/16)
			tg = clamp(tg + quantErrorCurr[x+1][1]/16)
			tb = clamp(tb + quantErrorCurr[x+1][2]/16)
			ta = clamp(ta + quantErrorCurr[x+1][3]/16)

			out.R = uint8(tr >> 8)
			out.G = uint8(tg >> 8)
			out.B = uint8(tb >> 8)
			out.A = uint8(ta >> 8)
			nrgba.Set(x, y, &out)

			tr -= int32(out.R) << 8
			tg -= int32(out.G) << 8
			tb -= int32(out.B) << 8
			ta -= int32(out.A) << 8

			// Propagate the Floyd-Steinberg quantization error.
			quantErrorNext[x+0][0] += tr * 3
			quantErrorNext[x+0][1] += tg * 3
			quantErrorNext[x+0][2] += tb * 3
			quantErrorNext[x+0][3] += ta * 3
			quantErrorNext[x+1][0] += tr * 5
			quantErrorNext[x+1][1] += tg * 5
			quantErrorNext[x+1][2] += tb * 5
			quantErrorNext[x+1][3] += ta * 5
			quantErrorNext[x+2][0] += tr * 1
			quantErrorNext[x+2][1] += tg * 1
			quantErrorNext[x+2][2] += tb * 1
			quantErrorNext[x+2][3] += ta * 1
			quantErrorCurr[x+2][0] += tr * 7
			quantErrorCurr[x+2][1] += tg * 7
			quantErrorCurr[x+2][2] += tb * 7
			quantErrorCurr[x+2][3] += ta * 7
		}

		// Recycle the quantization error buffers.
		quantErrorCurr, quantErrorNext = quantErrorNext, quantErrorCurr
		for i := range quantErrorNext {
			quantErrorNext[i] = [4]int32{}
		}
	}
	return nrgba
}

func quantize(in image.Image) (out *image.Paletted) {
	bounds := in.Bounds()
	switch t := in.(type) {
	case *image.Paletted:
		return t
	case *image.NRGBA:
		return imageQuant(t)
	case *image.NRGBA64:
		nrgba := floydSteinberg(t)
		return imageQuant(nrgba)
	default:
		// Convert to NRGBA from whatever it is
		nrgba := image.NewNRGBA(image.Rect(0, 0, bounds.Max.X, bounds.Max.Y))
		draw.Draw(nrgba, nrgba.Bounds(), in, image.Pt(0, 0), draw.Src)
		return imageQuant(nrgba)
	}
}

func help() {
	fmt.Println("pixelSequencer v0.3 - (c)2016-2017 by Fredrik Lidström")
	fmt.Println("")
	fmt.Println("pixelSequencer diffuse <input.png> <output.png>")
	fmt.Println("   Floyd-Steinberg error diffuse image (NRGBA64 png -> NRGBA png)")
	fmt.Println("")
	fmt.Println("pixelSequencer quantize <input.png> <output.png>")
	fmt.Println("   Quantize single image (png -> Paletted 8-bit)")
	fmt.Println("")
	fmt.Println("pixelSequencer unquantize <input.png> <output.png>")
	fmt.Println("   Unquantize single image (Paletted 8-bit -> NRGBA png)")
	fmt.Println("")
	fmt.Println("pixelSequencer encode <input.png> <frame-count> <output.png>")
	fmt.Println("   Encode animation (vertical frame strip png -> Paletted 8-bit pixel strip):")
	fmt.Println("")
	fmt.Println("pixelSequencer decode <input.png> <frame-count> <output.png>")
	fmt.Println("   Decode animation image (Paletted 8-bit pixel strip -> vertical frame strip NRGBA png")
	fmt.Println("")
	os.Exit(-1)
}

func main() {
	if len(os.Args) < 2 {
		help()
	}

	command := os.Args[1]
	switch command {
	case "diffuse", "quantize", "unquantize":
		if len(os.Args) < 3 {
			help()
		}
	case "encode", "decode":
		if len(os.Args) < 5 {
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
	case "diffuse":
		switch t := inputImage.(type) {
		case *image.NRGBA64:
			outputImage = floydSteinberg(t)
		default:
			fmt.Printf("No error diffusion needed")
			outputImage = inputImage
		}
	case "quantize":
		outputImage = quantize(inputImage)
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

		frameStripImage := quantize(inputImage)
		fmt.Println("Encoding horizontal pixel strip from vertical frame strip")

		// Re-arrange all frame pixels in a sequence
		sequenceImage := image.NewPaletted(image.Rect(0, 0, frameC*frameW, frameH), frameStripImage.Palette)
		{
			x := 0
			f := 0
			for _, c := range frameStripImage.Pix {
				sequenceImage.Pix[f+x] = c
				x += frameC
				if x >= frameC*frameW*frameH {
					f++
					x %= frameC * frameW * frameH
				}
			}
		}
		outputImage = sequenceImage

	case "decode":
		frameC, err := strconv.Atoi(os.Args[3])
		panicOn(err)
		frameW := (inputW / frameC)
		if frameW*frameC != inputW {
			panic(fmt.Sprintf("Image width %d is not even divisable by frame count %d", inputW, frameC))
		}
		frameH := inputH

		fmt.Printf("Number of frames: %d (%dx%d)\n", frameC, frameW, frameH)

		fmt.Println("Decoding horizontal pixel strip to vertical frame strip")
		strip := image.NewNRGBA(image.Rect(0, 0, frameW, frameH*frameC))
		{
			sequenceImage := inputImage.(*image.Paletted)
			i := 0
			for y := 0; y < frameH; y++ {
				for x := 0; x < frameW; x++ {
					for f := 0; f < frameC; f++ {
						c := sequenceImage.Palette[sequenceImage.Pix[i]]
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
