// +build !linux

package glhl

import "errors"

type platformContext struct{}

func initPlatform(ctx *Context) error {
	return errors.New("No platform-specific init")
}
func (platformContext) Destroy() {}
