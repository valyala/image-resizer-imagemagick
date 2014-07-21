// Copyright 2013 Herbert G. Fischer. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package imagick

/*
#cgo pkg-config: MagickWand MagickCore
#include <wand/MagickWand.h>
*/
import "C"

import (
	"unsafe"
)

type MagickWand struct {
	mw *C.MagickWand
}

func NewMagickWand() *MagickWand {
	return &MagickWand{C.NewMagickWand()}
}

func (mw *MagickWand) Destroy() {
	if mw.mw == nil {
		return
	}
	mw.mw = C.DestroyMagickWand(mw.mw)
	C.free(unsafe.Pointer(mw.mw))
	mw.mw = nil
}

func (mw *MagickWand) ResetIterator() {
	C.MagickResetIterator(mw.mw)
}

func (mw *MagickWand) relinquishMemory(ptr unsafe.Pointer) {
	C.MagickRelinquishMemory(ptr)
}

