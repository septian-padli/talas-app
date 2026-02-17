package media

import (
	"context"
	"errors"
	"mime/multipart"

	"github.com/cloudinary/cloudinary-go/v2"
	"github.com/cloudinary/cloudinary-go/v2/api/uploader"
	"github.com/septianpadli/talas/content-service/internal/config"
	"github.com/sirupsen/logrus"
)

// MediaUploader defines the contract for uploading media
type MediaUploader interface {
	Upload(ctx context.Context, file multipart.File, filename string, folder string) (string, error)
}

// CloudinaryUploader implementation
type CloudinaryUploader struct {
	cld *cloudinary.Cloudinary
	log *logrus.Logger
}

func NewCloudinaryUploader(cfg *config.Config, log *logrus.Logger) (*CloudinaryUploader, error) {
	cld, err := cloudinary.NewFromParams(cfg.CloudinaryCloudName, cfg.CloudinaryAPIKey, cfg.CloudinaryAPISecret)
	if err != nil {
		return nil, err
	}
	return &CloudinaryUploader{cld: cld, log: log}, nil
}

func (u *CloudinaryUploader) Upload(ctx context.Context, file multipart.File, filename string, folder string) (string, error) {
	useFilename := true
	uniqueFilename := true
	overwrite := false
	params := uploader.UploadParams{
		Folder:         folder,
		PublicID:       filename,
		UniqueFilename: &uniqueFilename,
		UseFilename:    &useFilename,
		Overwrite:      &overwrite,
	}
	uploadResult, err := u.cld.Upload.Upload(ctx, file, params)
	if err != nil {
		u.log.Errorf("Failed to upload file %s: %v", filename, err)
		return "", errors.New("failed to upload image")
	}
	return uploadResult.SecureURL, nil
}
