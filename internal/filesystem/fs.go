package filesystem

import "io/fs"

type FS interface {
	fs.ReadDirFS
	fs.ReadFileFS
	fs.StatFS
}
