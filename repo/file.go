package repo

import (
	"bufio"
	"cdp/config"
	"cdp/pkg/goutil"
	"context"
	"encoding/csv"
	"encoding/json"
	"google.golang.org/api/drive/v3"
	"google.golang.org/api/option"
	"io"
	"strings"
)

type FileRepo interface {
	CreateFile(ctx context.Context, parentID *string, fileName string, data io.Reader) (string, error)
	CreateFolder(ctx context.Context, folderName string) (string, error)
	DownloadFile(_ context.Context, fileID string) ([][]string, error)
	Close(ctx context.Context) error
}

type fileRepo struct {
	baseFolderID string
	adminEmail   string

	srv *drive.Service
}

func NewFileRepo(ctx context.Context, cfg config.GoogleDrive) (FileRepo, error) {
	b, err := json.Marshal(cfg.GoogleServiceAccount)
	if err != nil {
		return nil, err
	}

	srv, err := drive.NewService(ctx, option.WithCredentialsJSON(b))
	if err != nil {
		return nil, err
	}

	return &fileRepo{
		adminEmail:   cfg.AdminEmail,
		baseFolderID: cfg.BaseFolderID,
		srv:          srv,
	}, nil
}

func (r *fileRepo) CreateFolder(_ context.Context, folderName string) (string, error) {
	folder, err := r.srv.Files.Create(&drive.File{
		Name:     folderName,
		MimeType: "application/vnd.google-apps.folder",
	}).Do()
	if err != nil {
		return "", err
	}

	if err := r.addBasePermissions(folder.Id); err != nil {
		return "", err
	}

	return folder.Id, nil
}

func (r *fileRepo) CreateFile(_ context.Context, parentID *string, fileName string, data io.Reader) (string, error) {
	if parentID == nil {
		parentID = goutil.String(r.baseFolderID)
	}
	f := &drive.File{
		Name:    fileName,
		Parents: []string{*parentID},
	}

	file, err := r.srv.Files.Create(f).Media(data).Do()
	if err != nil {
		return "", err
	}
	return file.Id, err
}

func (r *fileRepo) addBasePermissions(fileID string) error {
	_, err := r.srv.Permissions.Create(fileID, &drive.Permission{
		Type:         "user",
		Role:         "writer",
		EmailAddress: r.adminEmail,
	}).Do()
	if err != nil {
		return err
	}

	return nil
}

func (r *fileRepo) DownloadFile(_ context.Context, fileID string) ([][]string, error) {
	resp, err := r.srv.Files.Get(fileID).Download()
	if err != nil {
		return nil, err
	}

	defer func() {
		_ = resp.Body.Close()
	}()

	scanner := bufio.NewScanner(resp.Body)

	var (
		records  [][]string
		isHeader = true
	)

	for scanner.Scan() {
		line := scanner.Text()

		if isHeader {
			isHeader = false
			continue
		}

		reader := csv.NewReader(strings.NewReader(line))
		row, err := reader.Read()
		if err != nil {
			return nil, err
		}

		records = append(records, row)
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return records, nil
}

func (r *fileRepo) Close(_ context.Context) error {
	return nil
}
