// +build !linux

package validate

// CheckLinux is a noop on this platform
func (v *Validator) CheckLinux() (errs error) {
	return nil
}
