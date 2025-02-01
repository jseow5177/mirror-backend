package handler

import (
	"cdp/entity"
)

type FileUpload struct {
	FileMeta *entity.FileMeta
}

func (f *FileUpload) SetFileMeta(m *entity.FileMeta) {
	f.FileMeta = m
}

func (f *FileUpload) GetFileName() string {
	if f != nil && f.FileMeta != nil && f.FileMeta.FileHeader != nil {
		return f.FileMeta.FileHeader.Filename
	}
	return ""
}
