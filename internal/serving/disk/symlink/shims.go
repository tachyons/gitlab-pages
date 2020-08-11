package filepath

import stdlib "path/filepath"

func volumeNameLen(s string) int { return 0 }

func IsAbs(path string) bool                   { return stdlib.IsAbs(path) }
func Clean(path string) string                 { return stdlib.Clean(path) }
func Join(elem ...string) string               { return stdlib.Join(elem...) }
func VolumeName(path string) string            { return stdlib.VolumeName(path) }
func EvalSymlinks(path string) (string, error) { return walkSymlinks(path) }

const Separator = stdlib.Separator
