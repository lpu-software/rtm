package analyzer

import (
	"image"
	"math"
)

// ImageMetrics contains computed statistical, structural, and security-relevant image comparison metrics.
type ImageMetrics struct {
	MSE                        float64 `json:"mse"`
	PSNR                       float64 `json:"psnr"`
	SSIM                       float64 `json:"ssim"`
	GlobalSimilarityScore      float64 `json:"global_similarity_score"` // 0.0 to 1.0
	BlackPixelRatio            float64 `json:"black_pixel_ratio"`       // 0.0 to 1.0 (RGB < 10)
	AlphaTransparentRatio      float64 `json:"alpha_transparent_ratio"` // 0.0 to 1.0 (A == 0)
	BlurLaplacianVariance      float64 `json:"blur_laplacian_variance"` // Higher = sharper
	ReferenceSharpness         float64 `json:"reference_sharpness"`
	SensitiveRegionSimilarity  float64 `json:"sensitive_region_similarity"`
	SensitiveRegionBlackRatio  float64 `json:"sensitive_region_black_ratio"`
	SensitiveRegionMaskDetected bool   `json:"sensitive_region_mask_detected"`
	QRMatrixDestroyed          bool    `json:"qr_matrix_destroyed"`
	FrameCounterDelta          int64   `json:"frame_counter_delta"`
	IsFrozen                   bool    `json:"is_frozen"`
}

// ComputeMetrics performs full image comparison between a reference image and a captured image.
func ComputeMetrics(refImg, capImg *image.RGBA, sensitiveBox, qrBox image.Rectangle, expectedFrame int64) *ImageMetrics {
	if refImg == nil || capImg == nil {
		return &ImageMetrics{
			MSE:                   math.MaxFloat64,
			PSNR:                  0,
			SSIM:                  0,
			GlobalSimilarityScore: 0,
		}
	}

	// 1. Rescale or crop captured image if dimensions differ
	alignedCap := alignAndCrop(capImg, refImg.Bounds().Dx(), refImg.Bounds().Dy())

	// 2. Compute MSE & PSNR
	mse := computeMSE(refImg, alignedCap)
	psnr := computePSNR(mse)

	// 3. Compute Structural Similarity Index (SSIM)
	ssim := computeSSIM(refImg, alignedCap)

	// 4. Black Pixel & Alpha Transparency Ratios
	blackRatio, alphaRatio := computeBlackAndAlphaRatio(alignedCap)

	// 5. Laplacian Variance for Blur Detection
	refSharpness := computeLaplacianVariance(refImg)
	capSharpness := computeLaplacianVariance(alignedCap)

	// 6. Sensitive Sub-Region Metrics
	sensSim, sensBlack, sensMask := computeSubRegionMetrics(refImg, alignedCap, sensitiveBox)

	// 7. QR Matrix Destruction Check
	qrDestroyed := checkQRMatrixDestroyed(refImg, alignedCap, qrBox)

	// 8. Global Similarity Score (Clamped 0.0 to 1.0)
	globalScore := math.Max(0.0, math.Min(1.0, ssim))

	return &ImageMetrics{
		MSE:                        mse,
		PSNR:                       psnr,
		SSIM:                       ssim,
		GlobalSimilarityScore:      globalScore,
		BlackPixelRatio:            blackRatio,
		AlphaTransparentRatio:      alphaRatio,
		BlurLaplacianVariance:      capSharpness,
		ReferenceSharpness:         refSharpness,
		SensitiveRegionSimilarity:  sensSim,
		SensitiveRegionBlackRatio:  sensBlack,
		SensitiveRegionMaskDetected: sensMask,
		QRMatrixDestroyed:          qrDestroyed,
		FrameCounterDelta:          0,
		IsFrozen:                   false,
	}
}

// alignAndCrop normalizes the captured image to match the reference dimensions.
func alignAndCrop(src *image.RGBA, targetW, targetH int) *image.RGBA {
	b := src.Bounds()
	if b.Dx() == targetW && b.Dy() == targetH {
		return src
	}

	dst := image.NewRGBA(image.Rect(0, 0, targetW, targetH))
	for y := 0; y < targetH; y++ {
		srcY := b.Min.Y + int(float64(y)*float64(b.Dy())/float64(targetH))
		if srcY >= b.Max.Y {
			srcY = b.Max.Y - 1
		}
		for x := 0; x < targetW; x++ {
			srcX := b.Min.X + int(float64(x)*float64(b.Dx())/float64(targetW))
			if srcX >= b.Max.X {
				srcX = b.Max.X - 1
			}
			dst.SetRGBA(x, y, src.RGBAAt(srcX, srcY))
		}
	}
	return dst
}

// computeMSE calculates the Mean Squared Error between two RGBA images.
func computeMSE(imgA, imgB *image.RGBA) float64 {
	b := imgA.Bounds()
	var sumSq float64
	total := float64(b.Dx() * b.Dy() * 3)

	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			cA := imgA.RGBAAt(x, y)
			cB := imgB.RGBAAt(x, y)

			dr := float64(cA.R) - float64(cB.R)
			dg := float64(cA.G) - float64(cB.G)
			db := float64(cA.B) - float64(cB.B)

			sumSq += dr*dr + dg*dg + db*db
		}
	}

	return sumSq / total
}

func computePSNR(mse float64) float64 {
	if mse <= 0.0001 {
		return 100.0 // Practically infinite
	}
	return 10.0 * math.Log10((255.0*255.0)/mse)
}

// computeSSIM calculates the Structural Similarity Index between two images.
func computeSSIM(imgA, imgB *image.RGBA) float64 {
	b := imgA.Bounds()
	w, h := b.Dx(), b.Dy()
	if w < 8 || h < 8 {
		return 1.0
	}

	// Constants to stabilize division with weak denominator
	const (
		c1 = (0.01 * 255) * (0.01 * 255)
		c2 = (0.03 * 255) * (0.03 * 255)
	)

	blockSize := 16
	var ssimSum float64
	var blockCount float64

	for by := 0; by <= h-blockSize; by += blockSize {
		for bx := 0; bx <= w-blockSize; bx += blockSize {
			var meanA, meanB float64
			var varA, varB, covAB float64
			n := float64(blockSize * blockSize)

			// Step 1: Compute Means (using Luminance Y = 0.299R + 0.587G + 0.114B)
			for y := 0; y < blockSize; y++ {
				for x := 0; x < blockSize; x++ {
					cA := imgA.RGBAAt(bx+x, by+y)
					cB := imgB.RGBAAt(bx+x, by+y)
					yA := 0.299*float64(cA.R) + 0.587*float64(cA.G) + 0.114*float64(cA.B)
					yB := 0.299*float64(cB.R) + 0.587*float64(cB.G) + 0.114*float64(cB.B)
					meanA += yA
					meanB += yB
				}
			}
			meanA /= n
			meanB /= n

			// Step 2: Compute Variances and Covariance
			for y := 0; y < blockSize; y++ {
				for x := 0; x < blockSize; x++ {
					cA := imgA.RGBAAt(bx+x, by+y)
					cB := imgB.RGBAAt(bx+x, by+y)
					yA := 0.299*float64(cA.R) + 0.587*float64(cA.G) + 0.114*float64(cA.B)
					yB := 0.299*float64(cB.R) + 0.587*float64(cB.G) + 0.114*float64(cB.B)

					diffA := yA - meanA
					diffB := yB - meanB

					varA += diffA * diffA
					varB += diffB * diffB
					covAB += diffA * diffB
				}
			}
			varA /= (n - 1)
			varB /= (n - 1)
			covAB /= (n - 1)

			// SSIM formula
			num := (2*meanA*meanB + c1) * (2*covAB + c2)
			den := (meanA*meanA + meanB*meanB + c1) * (varA + varB + c2)
			blockSSIM := num / den

			ssimSum += blockSSIM
			blockCount++
		}
	}

	if blockCount == 0 {
		return 0
	}
	return ssimSum / blockCount
}

// computeBlackAndAlphaRatio calculates percentage of black (<10,10,10) and transparent (A=0) pixels.
func computeBlackAndAlphaRatio(img *image.RGBA) (float64, float64) {
	b := img.Bounds()
	total := float64(b.Dx() * b.Dy())
	if total == 0 {
		return 0, 0
	}

	var blackCount, alphaCount float64
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			c := img.RGBAAt(x, y)
			if c.A < 10 {
				alphaCount++
			}
			if c.R < 10 && c.G < 10 && c.B < 10 {
				blackCount++
			}
		}
	}

	return blackCount / total, alphaCount / total
}

// computeLaplacianVariance calculates the variance of the 3x3 discrete Laplacian filter.
func computeLaplacianVariance(img *image.RGBA) float64 {
	b := img.Bounds()
	w, h := b.Dx(), b.Dy()
	if w < 3 || h < 3 {
		return 0
	}

	var lapSum, lapSumSq float64
	var count float64

	for y := 1; y < h-1; y++ {
		for x := 1; x < w-1; x++ {
			// Discrete Laplacian kernel: [0, 1, 0; 1, -4, 1; 0, 1, 0]
			cCenter := img.RGBAAt(x, y)
			cUp := img.RGBAAt(x, y-1)
			cDown := img.RGBAAt(x, y+1)
			cLeft := img.RGBAAt(x-1, y)
			cRight := img.RGBAAt(x+1, y)

			lumCenter := 0.299*float64(cCenter.R) + 0.587*float64(cCenter.G) + 0.114*float64(cCenter.B)
			lumUp := 0.299*float64(cUp.R) + 0.587*float64(cUp.G) + 0.114*float64(cUp.B)
			lumDown := 0.299*float64(cDown.R) + 0.587*float64(cDown.G) + 0.114*float64(cDown.B)
			lumLeft := 0.299*float64(cLeft.R) + 0.587*float64(cLeft.G) + 0.114*float64(cLeft.B)
			lumRight := 0.299*float64(cRight.R) + 0.587*float64(cRight.G) + 0.114*float64(cRight.B)

			lap := lumUp + lumDown + lumLeft + lumRight - 4.0*lumCenter
			lapSum += lap
			lapSumSq += lap * lap
			count++
		}
	}

	if count == 0 {
		return 0
	}
	mean := lapSum / count
	variance := (lapSumSq / count) - (mean * mean)
	return math.Max(0, variance)
}

// computeSubRegionMetrics evaluates sub-region metrics specifically for the sensitive box.
func computeSubRegionMetrics(refImg, capImg *image.RGBA, box image.Rectangle) (float64, float64, bool) {
	if box.Dx() <= 0 || box.Dy() <= 0 {
		return 1.0, 0.0, false
	}

	subRef := cropSubImage(refImg, box)
	subCap := cropSubImage(capImg, box)

	sim := computeSSIM(subRef, subCap)
	blackRatio, _ := computeBlackAndAlphaRatio(subCap)

	// Detect privacy overlay banner (amber/hazard pattern or dark veil over sensitive box)
	isMask := sim < 0.40 && blackRatio < 0.90
	return sim, blackRatio, isMask
}

func checkQRMatrixDestroyed(refImg, capImg *image.RGBA, qrBox image.Rectangle) bool {
	if qrBox.Dx() <= 0 || qrBox.Dy() <= 0 {
		return false
	}
	subRef := cropSubImage(refImg, qrBox)
	subCap := cropSubImage(capImg, qrBox)

	sim := computeSSIM(subRef, subCap)
	return sim < 0.30
}

func cropSubImage(src *image.RGBA, box image.Rectangle) *image.RGBA {
	dst := image.NewRGBA(image.Rect(0, 0, box.Dx(), box.Dy()))
	for y := 0; y < box.Dy(); y++ {
		for x := 0; x < box.Dx(); x++ {
			srcX := box.Min.X + x
			srcY := box.Min.Y + y
			if srcX < src.Bounds().Max.X && srcY < src.Bounds().Max.Y {
				dst.SetRGBA(x, y, src.RGBAAt(srcX, srcY))
			}
		}
	}
	return dst
}
