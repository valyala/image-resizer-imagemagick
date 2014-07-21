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

type PixelWand struct {
	pw *C.PixelWand
}

func NewPixelWand() *PixelWand {
	return &PixelWand{C.NewPixelWand()}
}

func (pw *PixelWand) Destroy() {
	if pw.pw == nil {
		return
	}
	pw.pw = C.DestroyPixelWand(pw.pw)
	C.free(unsafe.Pointer(pw.pw))
	pw.pw = nil
}

func (pw *PixelWand) SetColor(color string) bool {
	cscolor := C.CString(color)
	defer C.free(unsafe.Pointer(cscolor))
	return 1 == int(C.PixelSetColor(pw.pw, cscolor))
}
