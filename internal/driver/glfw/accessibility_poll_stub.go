//go:build !accessibility || !windows

package glfw

func (*window) pollAccessibility() {}
