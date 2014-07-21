// Copyright 2013 Herbert G. Fischer. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package imagick

/*
#include <wand/MagickWand.h>
*/
import "C"

import (
	"errors"
	"unsafe"
)

func (mw *MagickWand) AnnotateImage(drawingWand *DrawingWand, x, y, angle float64, text string) error {
	cstext := C.CString(text)
	defer C.free(unsafe.Pointer(cstext))
	C.MagickAnnotateImage(mw.mw, drawingWand.dw, C.double(x), C.double(y), C.double(angle), cstext)
	return mw.GetLastError()
}

func (mw *MagickWand) GetImageBlob() []byte {
	clen := C.size_t(0)
	csblob := C.MagickGetImageBlob(mw.mw, &clen)
	defer mw.relinquishMemory(unsafe.Pointer(csblob))
	return C.GoBytes(unsafe.Pointer(csblob), C.int(clen))
}

func (mw *MagickWand) GetImageFormat() string {
	return C.GoString(C.MagickGetImageFormat(mw.mw))
}

func (mw *MagickWand) GetImageHeight() uint {
	return uint(C.MagickGetImageHeight(mw.mw))
}

func (mw *MagickWand) GetImageWidth() uint {
	return uint(C.MagickGetImageWidth(mw.mw))
}

func (mw *MagickWand) ReadImageBlob(blob []byte) error {
	if len(blob) == 0 {
		return errors.New("zero-length blob not permitted")
	}
	C.MagickReadImageBlob(mw.mw, unsafe.Pointer(&blob[0]), C.size_t(len(blob)))
	return mw.GetLastError()
}

func (mw *MagickWand) SetImageCompressionQuality(quality uint) error {
	C.MagickSetImageCompressionQuality(mw.mw, C.size_t(quality))
	return mw.GetLastError()
}

func (mw *MagickWand) SharpenImage(radius, sigma float64) error {
	C.MagickSharpenImage(mw.mw, C.double(radius), C.double(sigma))
	return mw.GetLastError()
}

func (mw *MagickWand) StripImage() error {
	C.MagickStripImage(mw.mw)
	return mw.GetLastError()
}

func (mw *MagickWand) ThumbnailImage(cols, rows uint) error {
	C.MagickThumbnailImage(mw.mw, C.size_t(cols), C.size_t(rows))
	return mw.GetLastError()
}

