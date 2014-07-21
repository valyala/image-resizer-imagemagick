// Copyright 2013 Herbert G. Fischer. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package imagick

/*
#include <wand/MagickWand.h>
*/
import "C"
import (
	"unsafe"
)

type DrawingWand struct {
	dw *C.DrawingWand
}

func NewDrawingWand() *DrawingWand {
	return &DrawingWand{C.NewDrawingWand()}
}

func (dw *DrawingWand) Destroy() {
	if dw.dw == nil {
		return
	}
	dw.dw = C.DestroyDrawingWand(dw.dw)
	C.free(unsafe.Pointer(dw.dw))
	dw.dw = nil

}

func (dw *DrawingWand) SetFillColor(fillWand *PixelWand) {
	C.DrawSetFillColor(dw.dw, fillWand.pw)
}

func (dw *DrawingWand) SetFont(fontName string) error {
	csFontName := C.CString(fontName)
	defer C.free(unsafe.Pointer(csFontName))
	C.DrawSetFont(dw.dw, csFontName)
	return dw.GetLastError()
}

func (dw *DrawingWand) SetFontSize(pointSize float64) {
	C.DrawSetFontSize(dw.dw, C.double(pointSize))
}

func (dw *DrawingWand) SetFontStyle(style StyleType) {
	C.DrawSetFontStyle(dw.dw, C.StyleType(style))
}

func (dw *DrawingWand) SetFontWeight(fontWeight uint) {
	C.DrawSetFontWeight(dw.dw, C.size_t(fontWeight))
}

func (dw *DrawingWand) SetGravity(gravity GravityType) {
	C.DrawSetGravity(dw.dw, C.GravityType(gravity))
}

func (dw *DrawingWand) SetStrokeColor(strokeWand *PixelWand) {
	C.DrawSetStrokeColor(dw.dw, strokeWand.pw)
}

func (dw *DrawingWand) SetStrokeWidth(width float64) {
	C.DrawSetStrokeWidth(dw.dw, C.double(width))
}

