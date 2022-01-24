# pixelSequencer
Command line utility for encoding and decoding pixel strips.
```
pixelSequencer v0.6 - (c) 2016-2022 by Fredrik Lidström

pixelSequencer diffuse <input.png> <output.png>
   Floyd-Steinberg error diffuse image (NRGBA64 png -> NRGBA png)

pixelSequencer quantize <input.png> <output.png>
   Quantize single image (png -> Paletted 8-bit)

pixelSequencer unquantize <input.png> <output.png>
   Unquantize single image (Paletted 8-bit -> NRGBA png)

pixelSequencer encode <input.png> <frame-count> <output.png>
   Encode animation (vertical frame strip png -> Paletted 8-bit pixel strip):

pixelSequencer decode <input.png> <frame-count> <output.png>
   Decode animation image (Paletted 8-bit pixel strip -> vertical frame strip NRGBA png
```
