package handler

import (
	"cdp/entity"
	"mime/multipart"
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

func (f *FileUpload) GetFile() multipart.File {
	if f != nil && f.FileMeta != nil && f.FileMeta.File != nil {
		return f.FileMeta.File
	}
	return nil
}
