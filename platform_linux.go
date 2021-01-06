package glhl

/*
#cgo LDFLAGS: -lEGL -lgbm
#include <EGL/egl.h>
#include <gbm.h>
*/
import "C"

import (
	"errors"
	"fmt"
	"os"
	"strings"
	"unsafe"
)

type platformContext struct {
	gbm  *C.struct_gbm_device
	gbmf *os.File
}

func initPlatform(ctx *Context) error {
	ext := C.eglQueryString(C.EGLDisplay(C.EGL_NO_DISPLAY), C.EGL_EXTENSIONS)
	if ext == nil || !strings.Contains(C.GoString(ext), "EGL_MESA_platform_gbm") {
		return ErrUnsupported
	}

	var err error
	ctx.gbmf, err = os.OpenFile("/dev/dri/card0", os.O_RDWR, 0) // FIXME: don't indiscriminately use card0
	if err != nil {
		return err
	}
	ctx.gbm = C.gbm_create_device(C.int(ctx.gbmf.Fd()))
	if ctx.gbm == nil {
		ctx.gbmf.Close()
		return ErrGBM
	}

	ctx.dpy = C.eglGetPlatformDisplay(egl_PLATFORM_GBM_MESA, unsafe.Pointer(ctx.gbm), nil)
	if ctx.dpy == C.EGLDisplay(C.EGL_NO_DISPLAY) {
		ctx.platformContext.Destroy()
		return ErrNoDisplay
	}

	if C.eglInitialize(ctx.dpy, nil, nil) == 0 {
		return fmt.Errorf("eglInitialize: %w", eglError())
	}

	return nil
}

func (ctx platformContext) Destroy() {
	if ctx.gbm != nil {
		C.gbm_device_destroy(ctx.gbm)
		ctx.gbmf.Close()

		ctx.gbm = nil
		ctx.gbmf = nil
	}
}

const egl_PLATFORM_GBM_MESA C.EGLenum = 0x31D7

var ErrGBM = errors.New("GBM error")
