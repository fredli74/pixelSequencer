package main

/*
  #cgo !windows CFLAGS: -Ilibimagequant
	#cgo !windows LDFLAGS: libimagequant_unix.a
  #cgo  windows CFLAGS: -fopenmp -Ilibimagequant
	#cgo  windows LDFLAGS: -fopenmp libimagequant_win.a
	#include "libimagequant.h"
*/
import "C"

import (
	"fmt"
	"image"
	"image/color"
	"image/color/palette"
	"unsafe"
)

func imageQuant(src *image.NRGBA) (out *image.Paletted) {
	fmt.Printf("Quantizing image ... ")
	bounds := src.Bounds()
	w := bounds.Max.X
	h := bounds.Max.Y

	// Setup imagequant attributes
	quantAttributes := C.liq_attr_create()
	if quantAttributes == nil {
		panic("Unable to initialize quantize attributes, imagequant library returned null")
	}
	defer C.liq_attr_destroy(quantAttributes)

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
