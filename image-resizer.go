package main

import (
	"./imagick/imagick"
	"flag"
	"fmt"
	"github.com/mitchellh/goamz/aws"
	"github.com/mitchellh/goamz/s3"
	"github.com/valyala/ybc/bindings/go/ybc"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"strconv"
	"strings"
)

var (
	defaultCompressionQuality  = flag.Uint("defaultCompressionQuality", 75, "Default compression quality for images. It may be overrided by compressionQuality parameter")
	listenAddr                 = flag.String("listenAddr", ":8081", "TCP address to listen to")
	maxImageSize               = flag.Int64("maxImageSize", 10*1024*1024, "The maximum image size which can be read from imageUrl")
	maxUpstreamCacheItemsCount = flag.Int("maxCachedImagesCount", 10*1000, "The maximum number of images the resizer can cache from upstream servers. Increase this value for saving more upstream bandwidth")
	maxUpstreamCacheSize       = flag.Int("maxUpstreamCacheSize", 100, "The maximum total size in MB of images the resizer cache cache from upstream servers. Increase this value for saving more upstream bandwidth")
	s3AccessKey                = flag.String("s3AccessKey", "foobar", "Access key for Amazon S3")
	s3BucketName               = flag.String("s3Bucket", "bucket", "Amazon S3 bucket for loading images")
	s3Region                   = flag.String("s3Region", "eu-west-1", "Amazon region to route S3 requests to")
	s3SecretKey                = flag.String("s3SecretKey", "foobaz", "Secret key for Amazon S3")
	upstreamCacheFilename      = flag.String("upstreamCacheFilename", "", "Path to cache file for images loaded from upstream. Leave blank for anonymous non-persistent cache")
)

var (
	s3Bucket      *s3.Bucket
	upstreamCache ybc.Cacher
)

func main() {
	flag.Parse()

	imagick.Initialize()
	defer imagick.Terminate()

	upstreamCache = openUpstreamCache()
	defer upstreamCache.Close()

	s3Bucket = getS3Bucket()

	if err := http.ListenAndServe(*listenAddr, http.HandlerFunc(serveHTTP)); err != nil {
		logFatal("Error when starting or running http server: %v", err)
	}
}

func serveHTTP(w http.ResponseWriter, r *http.Request) {
	if r.RequestURI == "/favicon.ico" {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	imageUrl, width, height, compressionQuality, sharpFactor, bottomAnnotation, centerAnnotation := getImageParams(r)
	if len(imageUrl) == 0 {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	mw := imagick.NewMagickWand()
	defer mw.Destroy()

	if !loadImage(r, mw, imageUrl) {
		w.WriteHeader(http.StatusServiceUnavailable)
		return
	}

	var shouldResize bool
	width, height, shouldResize = adjustImageDimensions(mw, width, height)
	if shouldResize {
		if err := mw.ThumbnailImage(width, height); err != nil {
			logRequestError(r, "Error when thumbnailing the image obtained from imageUrl=%v: %v", imageUrl, err)
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
	}
	annotateImage(mw, bottomAnnotation, imagick.GRAVITY_SOUTH)
	annotateImage(mw, centerAnnotation, imagick.GRAVITY_CENTER)

	if sharpFactor >= 0 {
		mw.SharpenImage(0, sharpFactor)
	}
	if compressionQuality == 0 {
		compressionQuality = *defaultCompressionQuality
	}
	mw.SetImageCompressionQuality(compressionQuality)
	mw.StripImage()

	if !sendResponse(w, r, mw, imageUrl) {
		// w.WriteHeader() is skipped intentionally here, since the response may be already partially created.
		return
	}
	logRequestMessage(r, "SUCCESS")
}

func annotateImage(mw *imagick.MagickWand, annotation string, gravity imagick.GravityType) {
	if annotation == "" {
		return
	}

	dw := imagick.NewDrawingWand()
	defer dw.Destroy()
	pw := imagick.NewPixelWand()
	defer pw.Destroy()

	fontSize := getFontSize(mw, annotation)
	dw.SetFont("Verdana")
	dw.SetFontSize(fontSize)
	dw.SetFontWeight(100)
	dw.SetFontStyle(imagick.STYLE_NORMAL)
	dw.SetGravity(gravity)

	pw.SetColor("#ffffff80")
	dw.SetFillColor(pw)

	if fontSize > 20 {
		pw.SetColor("#00000050")
		dw.SetStrokeColor(pw)
		dw.SetStrokeWidth(1.0)
	}

	mw.AnnotateImage(dw, 0, 0, 0, annotation)
}

func getFontSize(mw *imagick.MagickWand, text string) float64 {
	fontSize := float64(80.0)
	fontSizeByWidth := float64(mw.GetImageWidth()) / float64(len(text)) / 0.55
	if fontSizeByWidth < fontSize {
		fontSize = fontSizeByWidth
	}
	fontSizeByHeight := float64(mw.GetImageHeight()) / 1.2
	if fontSizeByHeight < fontSize {
		fontSize = fontSizeByHeight
	}
	if fontSize < 10 {
		fontSize = 10
	}
	return fontSize
}

func adjustImageDimensions(mw *imagick.MagickWand, width, height uint) (uint, uint, bool) {
	if width == 0 && height == 0 {
		return 0, 0, false
	}

	if width == 0 {
		width = height
	} else if height == 0 {
		height = width
	}

	ow := mw.GetImageWidth()
	oh := mw.GetImageHeight()

	if ow <= width && oh <= height {
		return ow, oh, false
	}

	if ow > width {
		oh = uint(float64(oh) * float64(width) / float64(ow))
		ow = width
	}

	if oh > height {
		ow = uint(float64(ow) * float64(height) / float64(oh))
		oh = height
	}

	return ow, oh, true
}

func sendResponse(w http.ResponseWriter, r *http.Request, mw *imagick.MagickWand, imageUrl string) bool {
	mw.ResetIterator()
	blob := mw.GetImageBlob()
	format := mw.GetImageFormat()
	contentType := fmt.Sprintf("image/%s", strings.ToLower(format))
	w.Header().Set("Content-Type", contentType)
	if _, err := w.Write(blob); err != nil {
		logRequestError(r, "Cannot send image from imageUrl=%v to client: %v", imageUrl, err)
		return false
	}
	return true
}

func getImageParams(r *http.Request) (imageUrl string, width, height uint, compressionQuality uint, sharpFactor float64, bottomAnnotation, centerAnnotation string) {
	imageUrl = r.FormValue("imageUrl")
	if imageUrl == "" {
		imageUrl = r.URL.Path[1:]
		parts := strings.SplitN(imageUrl, "_", 4)
		if len(parts) != 4 {
			logRequestError(r, "imageUrl=%s must contain at least four parts delimited by '_'")
			imageUrl = ""
			return
		}
		imageUrl = fmt.Sprintf("s3:%s_%s", parts[0], parts[3])
		width = parseUint(r, "width", parts[1][1:])
		height = parseUint(r, "height", parts[2][1:])
	} else {
		width = getUint(r, "width")
		height = getUint(r, "height")
	}
	compressionQuality = getUint(r, "compressionQuality")
	sharpFactor = getFloat64(r, "sharpFactor")
	bottomAnnotation = r.FormValue("bottomAnnotation")
	centerAnnotation = r.FormValue("centerAnnotation")
	return
}

func getFloat64(r *http.Request, key string) float64 {
	v := r.FormValue(key)
	if len(v) == 0 {
		return 0.0
	}
	f, err := strconv.ParseFloat(v, 64)
	if err != nil {
		logRequestError(r, "Cannot parse %s=%v: %v", key, v, err)
		return 0.0
	}
	return f
}

func getUint(r *http.Request, key string) uint {
	v := r.FormValue(key)
	return parseUint(r, key, v)
}

func parseUint(r *http.Request, k, v string) uint {
	if len(v) == 0 {
		return 0
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		logRequestError(r, "Cannot parse %s=%v: %v", k, v, err)
		return 0
	}
	if n < 0 {
		logRequestError(r, "%s=%v must be positive", k, v)
		return 0
	}
	return uint(n)
}

func loadImage(r *http.Request, mw *imagick.MagickWand, imageUrl string) bool {
	blob := getImageBlob(r, imageUrl)
	if blob == nil {
		return false
	}
	if err := mw.ReadImageBlob(blob); err != nil {
		logRequestError(r, "Cannot parse image from imageUrl=%v: %v", imageUrl, err)
		return false
	}
	return true
}

func getImageBlob(r *http.Request, imageUrl string) []byte {
	blob, err := upstreamCache.Get([]byte(imageUrl))
	if err == nil {
		return blob
	}

	if err != ybc.ErrCacheMiss {
		logFatal("Unexpected error when reading data from upstream cache under the key=%v: %v", imageUrl, err)
	}

	if strings.Index(imageUrl, "s3:") == 0 {
		key := imageUrl[3:]
		if blob, err = s3Bucket.Get(key); err != nil {
			logRequestError(r, "Cannot fetch image by key=[%s] from Amazon S3: %v", key, err)
			return nil
		}
	} else {
		resp, err := http.Get(imageUrl)
		if err != nil {
			logRequestError(r, "Cannot load image from imageUrl=%v: %v", imageUrl, err)
			return nil
		}
		defer resp.Body.Close()
		if resp.StatusCode != 200 {
			logRequestError(r, "Unexpected StatusCode=%d returned from imageUrl=%v", resp.StatusCode, imageUrl)
			return nil
		}
		if blob, err = ioutil.ReadAll(io.LimitReader(resp.Body, *maxImageSize)); err != nil {
			logRequestError(r, "Error when reading image body from imageUrl=%v: %v", imageUrl, err)
			return nil
		}
	}

	if err = upstreamCache.Set([]byte(imageUrl), blob, ybc.MaxTtl); err != nil {
		if err == ybc.ErrNoSpace {
			logRequestError(r, "No enough space for storing image obtained from imageUrl=%v into upstream cache", imageUrl)
		} else {
			logFatal("Unexpected error when storing image under the key=%v in upstream cache: %v", imageUrl, err)
		}
	}
	return blob
}

func getS3Bucket() *s3.Bucket {
	auth := aws.Auth{
		AccessKey: *s3AccessKey,
		SecretKey: *s3SecretKey,
	}
	region, ok := aws.Regions[*s3Region]
	if !ok {
		logFatal("Unknown s3Region: %s", s3Region)
	}
	connection := s3.New(auth, region)
	return connection.Bucket(*s3BucketName)
}

func openUpstreamCache() ybc.Cacher {
	config := ybc.Config{
		MaxItemsCount: ybc.SizeT(*maxUpstreamCacheItemsCount),
		DataFileSize:  ybc.SizeT(*maxUpstreamCacheSize) * ybc.SizeT(1024*1024),
	}

	var err error
	var cache ybc.Cacher

	if *upstreamCacheFilename != "" {
		config.DataFile = *upstreamCacheFilename + ".cdn-booster.data"
		config.IndexFile = *upstreamCacheFilename + ".cdn-booster.index"
	}
	cache, err = config.OpenCache(true)
	if err != nil {
		logFatal("Cannot open cache for upstream images: %v", err)
	}
	return cache
}

func logRequestError(r *http.Request, format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	logRequestMessage(r, "ERROR: %s", msg)
}

func logRequestMessage(r *http.Request, format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	logMessage("%s - %s - %s - %s. %s", r.RemoteAddr, r.RequestURI, r.Referer(), r.UserAgent(), msg)
}

func logMessage(format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	log.Printf("%s\n", msg)
}

func logFatal(format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	log.Fatalf("%s\n", msg)
}
