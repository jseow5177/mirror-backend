package repo

import (
	"cdp/config"
	"context"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	"io"
	"time"
)

type FileRepo interface {
	Upload(ctx context.Context, key string, f io.Reader) (string, error)
	Download(ctx context.Context, key string) ([]byte, error)
	Close(ctx context.Context) error
}

type fileRepo struct {
	bucket            string
	uploader          *s3manager.Uploader
	downloader        *s3manager.Downloader
	expirationSeconds int64
}

func NewFileRepo(_ context.Context, s3Cfg config.S3) FileRepo {
	// start s3 client
	//sess := session.Must(session.NewSession(&aws.Config{
	//	Region:      goutil.String(s3Cfg.Region),
	//	Credentials: credentials.NewStaticCredentials(s3Cfg.AccessKeyID, s3Cfg.SecretAccessKey, ""),
	//}))

	return &fileRepo{
		//uploader:          s3manager.NewUploader(sess),
		//downloader:        s3manager.NewDownloader(sess),
		//bucket:            s3Cfg.Bucket,
		//expirationSeconds: s3Cfg.ExpirationSeconds,
	}
}

func (r *fileRepo) Upload(_ context.Context, key string, f io.Reader) (string, error) {
	expiry := time.Now().Add(time.Duration(r.expirationSeconds) * time.Second)

	res, err := r.uploader.Upload(&s3manager.UploadInput{
		Bucket:  aws.String(r.bucket),
		Key:     aws.String(key),
		Body:    f,
		Expires: &expiry,
	})
	if err != nil {
		return "", err
	}

	return res.Location, err
}

func (r *fileRepo) Download(_ context.Context, key string) ([]byte, error) {
	buf := aws.NewWriteAtBuffer(make([]byte, 0))

	_, err := r.downloader.Download(buf, &s3.GetObjectInput{
		Bucket: aws.String(r.bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

func (r *fileRepo) Close(_ context.Context) error {
	return nil
}
