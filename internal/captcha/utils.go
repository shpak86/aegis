package captcha

import (
	"image"
	"image/color"
	"math/rand"
)

func addUniformNoise(img image.Image, intensity int) *image.NRGBA {
	bounds := img.Bounds()
	noisyImg := image.NewNRGBA(bounds)

	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			originalColor := img.At(x, y)
			r, g, b, a := originalColor.RGBA()

			noiseR := rand.Intn(2*intensity+1) - intensity
			noiseG := rand.Intn(2*intensity+1) - intensity
			noiseB := rand.Intn(2*intensity+1) - intensity

			newR := clampInt(int(r>>8)+noiseR, 0, 255)
			newG := clampInt(int(g>>8)+noiseG, 0, 255)
			newB := clampInt(int(b>>8)+noiseB, 0, 255)

			noisyImg.Set(x, y, color.NRGBA{
				R: uint8(newR),
				G: uint8(newG),
				B: uint8(newB),
				A: uint8(a >> 8),
			})
		}
	}

	return noisyImg
}

func addNoise(img image.Image, intensity float64) *image.NRGBA {
	bounds := img.Bounds()
	noisyImg := image.NewNRGBA(bounds)

	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			originalColor := img.At(x, y)
			r, g, b, a := originalColor.RGBA()

			noiseR := rand.NormFloat64() * intensity
			noiseG := rand.NormFloat64() * intensity
			noiseB := rand.NormFloat64() * intensity

			newR := clamp(float64(r>>8)+noiseR, 0, 255)
			newG := clamp(float64(g>>8)+noiseG, 0, 255)
			newB := clamp(float64(b>>8)+noiseB, 0, 255)

			noisyImg.Set(x, y, color.NRGBA{
				R: uint8(newR),
				G: uint8(newG),
				B: uint8(newB),
				A: uint8(a >> 8),
			})
		}
	}

	return noisyImg
}

func clamp(value, min, max float64) float64 {
	if value < min {
		return min
	}
	if value > max {
		return max
	}
	return value
}

func clampInt(value, min, max int) int {
	if value < min {
		return min
	}
	if value > max {
		return max
	}
	return value
}
