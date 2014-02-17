package vcs

type FileType int

const (
	File FileType = iota
	Dir
)

func (t FileType) String() string {
	switch t {
	case File:
		return "File"
	case Dir:
		return "Dir"
	default:
		panic("unexpected")
	}
}
