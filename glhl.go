// glhl is a package to create headless OpenGL contexts, without the need for any attached display.
package glhl

/*
#cgo LDFLAGS: -lEGL
#include <stdlib.h>
#include <EGL/egl.h>
const EGLint _glhlConfigAttr[] = {
	EGL_CONFIG_CAVEAT, EGL_NONE,         // Require hardware acceleration
	EGL_CONFORMANT, EGL_OPENGL_BIT,      // Require OpenGL conformance
	EGL_RENDERABLE_TYPE, EGL_OPENGL_BIT, // Require OpenGL support
	EGL_NONE
};
int glhlNewContext(int major, int minor, int profile, _Bool debug, EGLDisplay *dpy, EGLContext *ctx, EGLContext sharedWith) {
	*dpy = eglGetDisplay(EGL_DEFAULT_DISPLAY);
	if (!eglInitialize(*dpy, NULL, NULL)) goto error;

	EGLint nconf;
	EGLConfig conf;
	if (!eglChooseConfig(*dpy, _glhlConfigAttr, &conf, 1, &nconf)) goto error;
	if (nconf < 1) return -1;

	if (!eglBindAPI(EGL_OPENGL_API)) goto error;
	EGLint ctxAttr[] = {
		EGL_CONTEXT_MAJOR_VERSION, major,
		EGL_CONTEXT_MINOR_VERSION, minor,
		EGL_CONTEXT_OPENGL_PROFILE_MASK, profile,
		EGL_CONTEXT_OPENGL_DEBUG, debug,
		EGL_NONE
	};
	*ctx = eglCreateContext(*dpy, conf, sharedWith, ctxAttr);

error:
	return eglGetError();
}
*/
import "C"

import (
	"fmt"
	"unsafe"
)

type Flag int

const (
	Core Flag = 1 << iota
	Compatibility
	Debug
)

type Context struct {
	dpy C.EGLDisplay
	ctx C.EGLContext
}

func NewContext(major, minor int, flags Flag) (Context, error) {
	return newContext(major, minor, flags, C.EGLContext(C.EGL_NO_CONTEXT))
}
func NewSharedContext(major, minor int, flags Flag, sharedWith Context) (Context, error) {
	return newContext(major, minor, flags, sharedWith.ctx)
}
func newContext(major, minor int, flags Flag, sharedWith C.EGLContext) (ctx Context, err error) {
	if flags == 0 {
		flags = Core
	}
	profile := C.int(0)
	if flags&Core != 0 {
		profile |= C.EGL_CONTEXT_OPENGL_CORE_PROFILE_BIT
	}
	if flags&Compatibility != 0 {
		profile |= C.EGL_CONTEXT_OPENGL_COMPATIBILITY_PROFILE_BIT
	}
	debug := C._Bool(flags&Debug != 0)

	code := C.glhlNewContext(C.int(major), C.int(minor), profile, debug, &ctx.dpy, &ctx.ctx, sharedWith)
	if code != C.EGL_SUCCESS {
		return Context{}, Error(code)
	}
	return ctx, nil
}

func (ctx Context) MakeCurrent() {
	noSurf := C.EGLSurface(C.EGL_NO_SURFACE)
	if C.eglMakeCurrent(ctx.dpy, noSurf, noSurf, ctx.ctx) == 0 {
		panic(Error(C.eglGetError()))
	}
}

func GetProcAddr(name string) unsafe.Pointer {
	cname := C.CString(name)
	defer C.free(unsafe.Pointer(cname))
	return unsafe.Pointer(C.eglGetProcAddress(cname))
}

type Error int

func (err Error) Error() string {
	switch err {
	case -1:
		str = "no matching config"
	case C.EGL_NOT_INITIALIZED:
		str = "not initialized"
	case C.EGL_BAD_ACCESS:
		str = "bad access"
	case C.EGL_BAD_ALLOC:
		str = "bad alloc"
	case C.EGL_BAD_ATTRIBUTE:
		str = "bad attribute"
	case C.EGL_BAD_CONFIG:
		str = "bad config"
	case C.EGL_BAD_CONTEXT:
		str = "bad context"
	case C.EGL_BAD_CURRENT_SURFACE:
		str = "bad current surface"
	case C.EGL_BAD_DISPLAY:
		str = "bad display"
	case C.EGL_BAD_MATCH:
		str = "bad match"
	case C.EGL_BAD_NATIVE_PIXMAP:
		str = "bad native pixmap"
	case C.EGL_BAD_NATIVE_WINDOW:
		str = "bad native window"
	case C.EGL_BAD_PARAMETER:
		str = "bad parameter"
	case C.EGL_BAD_SURFACE:
		str = "bad surface"
	case C.EGL_CONTEXT_LOST:
		str = "context lost"
	default:
		return fmt.Sprintf("unknown EGL error: %d", code)
	}
	return "EGL error: " + str
}
