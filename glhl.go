// glhl is a package to create headless OpenGL contexts, without the need for any attached display.
package glhl

/*
#cgo LDFLAGS: -lEGL -lgbm
#include <stdlib.h>
#include <string.h>
#include <EGL/egl.h>
#include <gbm.h>

int glhlMakeContextCurrent(EGLDisplay dpy, EGLContext ctx) {
	if (!eglBindAPI(EGL_OPENGL_API)) goto error;
	if (!eglMakeCurrent(dpy, EGL_NO_SURFACE, EGL_NO_SURFACE, ctx)) goto error;
	return EGL_SUCCESS;
error:
	return eglGetError();
}
*/
import "C"

import (
	"errors"
	"fmt"
	"os"
	"strings"
	"unsafe"
)

// Flags to NewContext and NewSharedContext.
// If neither Core nor Compatibility is provided, the default will be Core.
type Flag int

const (
	Core          Flag = 1 << iota // Allow core profile
	Compatibility                  // Allow compatibility profile
	Debug                          // Use debug context
)

// Context represents a headless OpenGL context
type Context struct {
	dpy  C.EGLDisplay
	ctx  C.EGLContext
	gbm  *C.struct_gbm_device
	gbmf *os.File
}

// NewContext creates a new context with the specified version and flags.
func NewContext(major, minor int, flags Flag) (Context, error) {
	return newContext(major, minor, flags)
}

const egl_PLATFORM_GBM_MESA C.EGLenum = 0x31D7

func initGeneric(ctx *Context) error {
	ctx.dpy = C.eglGetDisplay(C.EGL_DEFAULT_DISPLAY)
	if ctx.dpy == C.EGLDisplay(C.EGL_NO_DISPLAY) {
		return ErrNoDisplay
	}
	if C.eglInitialize(ctx.dpy, nil, nil) == 0 {
		return fmt.Errorf("eglInitialize: %w", eglError())
	}
	return nil
}

func initGBM(ctx *Context) error {
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
		C.gbm_device_destroy(ctx.gbm)
		ctx.gbmf.Close()
		return ErrNoDisplay
	}

	if C.eglInitialize(ctx.dpy, nil, nil) == 0 {
		return fmt.Errorf("eglInitialize: %w", eglError())
	}

	return nil
}

func newContext(major, minor int, flags Flag) (ctx Context, err error) {
	if err := initGeneric(&ctx); err != nil {
		if initGBM(&ctx) != nil {
			return Context{}, err
		}
	}

	var nconf C.EGLint
	var conf C.EGLConfig
	if C.eglChooseConfig(ctx.dpy, &configAttr[0], &conf, 1, &nconf) == 0 {
		return Context{}, fmt.Errorf("eglChooseConfig: %w", eglError())
	}
	if nconf < 1 {
		return Context{}, ErrNoConfig
	}

	if C.eglBindAPI(C.EGL_OPENGL_API) == 0 {
		return Context{}, fmt.Errorf("eglBindAPI: %w", eglError())
	}

	var profile C.EGLint
	if flags&Compatibility != 0 {
		profile |= C.EGL_CONTEXT_OPENGL_COMPATIBILITY_PROFILE_BIT
	}
	if flags&Core != 0 || profile == 0 {
		profile |= C.EGL_CONTEXT_OPENGL_CORE_PROFILE_BIT
	}
	ctxAttr := []C.EGLint{
		C.EGL_CONTEXT_MAJOR_VERSION, C.EGLint(major),
		C.EGL_CONTEXT_MINOR_VERSION, C.EGLint(minor),
		C.EGL_CONTEXT_OPENGL_PROFILE_MASK, profile,
	}
	if flags&Debug != 0 {
		ctxAttr = append(ctxAttr, C.EGL_CONTEXT_OPENGL_DEBUG, 1)
	}
	ctxAttr = append(ctxAttr, C.EGL_NONE)

	ctx.ctx = C.eglCreateContext(ctx.dpy, conf, C.EGLContext(C.EGL_NO_CONTEXT), &ctxAttr[0]) // TODO: shared contexts
	if err := eglError(); err != nil {
		return Context{}, fmt.Errorf("eglCreateContext: %w", err)
	}
	return ctx, nil
}

var configAttr = []C.EGLint{
	C.EGL_CONFIG_CAVEAT, C.EGL_NONE, // Require hardware acceleration
	C.EGL_CONFORMANT, C.EGL_OPENGL_BIT, // Require OpenGL conformance
	C.EGL_RENDERABLE_TYPE, C.EGL_OPENGL_BIT, // Require OpenGL support
	C.EGL_NONE,
}

// Destroy cleans up the state surrounding a context
func (ctx Context) Destroy() {
	if C.eglDestroyContext(ctx.dpy, ctx.ctx) == 0 {
		panic(Error(C.eglGetError()))
	}
	if ctx.gbm != nil {
		C.gbm_device_destroy(ctx.gbm)
		ctx.gbmf.Close()
	}
}

// MakeContextCurrent activates the context, making it the new current OpenGL context.
// gl.InitWithProcAddrFunc should be called with GetProcAddr after calling this function.
func (ctx Context) MakeContextCurrent() {
	code := C.glhlMakeContextCurrent(ctx.dpy, ctx.ctx)
	if code != C.EGL_SUCCESS {
		panic(Error(code))
	}
}

// Release deactivates the current context, making it available for use in other threads.
func Release() {
	if C.eglReleaseThread() == 0 {
		panic(Error(C.eglGetError()))
	}
}

// GetProcAddr gets the address of an OpenGL function. For use with gl.InitWithProcAddrFunc
func GetProcAddr(name string) unsafe.Pointer {
	cname := C.CString(name)
	defer C.free(unsafe.Pointer(cname))
	return unsafe.Pointer(C.eglGetProcAddress(cname))
}

func eglError() error {
	code := C.eglGetError()
	if code == C.EGL_SUCCESS {
		return nil
	} else {
		return Error(code)
	}
}

// Error represents context initialization error
type Error int

func (err Error) Error() string {
	switch err {
	case C.EGL_NOT_INITIALIZED:
		return "not initialized"
	case C.EGL_BAD_ACCESS:
		return "bad access"
	case C.EGL_BAD_ALLOC:
		return "bad alloc"
	case C.EGL_BAD_ATTRIBUTE:
		return "bad attribute"
	case C.EGL_BAD_CONFIG:
		return "bad config"
	case C.EGL_BAD_CONTEXT:
		return "bad context"
	case C.EGL_BAD_CURRENT_SURFACE:
		return "bad current surface"
	case C.EGL_BAD_DISPLAY:
		return "bad display"
	case C.EGL_BAD_MATCH:
		return "bad match"
	case C.EGL_BAD_NATIVE_PIXMAP:
		return "bad native pixmap"
	case C.EGL_BAD_NATIVE_WINDOW:
		return "bad native window"
	case C.EGL_BAD_PARAMETER:
		return "bad parameter"
	case C.EGL_BAD_SURFACE:
		return "bad surface"
	case C.EGL_CONTEXT_LOST:
		return "context lost"
	default:
		return fmt.Sprintf("unknown error: %d", err)
	}
}

var ErrNoDisplay = errors.New("No valid EGL display")
var ErrNoConfig = errors.New("No valid EGL config")
var ErrUnsupported = errors.New("Extension is unsupported")
var ErrGBM = errors.New("GBM error")
