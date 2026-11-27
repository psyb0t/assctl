//go:build !windows

package fonts

// registerFont does nothing outside Windows, where font directories are scanned instead.
func registerFont(string) error {
	return nil
}
