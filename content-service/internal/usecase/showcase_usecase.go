package usecase

import (
	"context"
	"errors"
	"mime/multipart"
	"strings"

	"github.com/cloudinary/cloudinary-go/v2"
	"github.com/cloudinary/cloudinary-go/v2/api/uploader"
	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
	"github.com/septianpadli/talas/content-service/internal/config"
	"github.com/septianpadli/talas/content-service/internal/entity"
	"github.com/septianpadli/talas/content-service/internal/repository"
	"github.com/sirupsen/logrus"
)

type ShowcaseUsecase interface {
	CreateShowcase(ctx context.Context, input *entity.CreateShowcaseRequest, files []*multipart.FileHeader, userID uuid.UUID) (*entity.Showcase, error)
}

type showcaseUsecase struct {
	repo     repository.ShowcaseRepository
	cfg      *config.Config
	log      *logrus.Logger
	validate *validator.Validate
}

func NewShowcaseUsecase(repo repository.ShowcaseRepository, cfg *config.Config, log *logrus.Logger) ShowcaseUsecase {
	return &showcaseUsecase{
		repo:     repo,
		cfg:      cfg,
		log:      log,
		validate: validator.New(),
	}
}

func (u *showcaseUsecase) CreateShowcase(ctx context.Context, input *entity.CreateShowcaseRequest, files []*multipart.FileHeader, userID uuid.UUID) (*entity.Showcase, error) {
	// 1. Validate Input
	if err := u.validate.Struct(input); err != nil {
		return nil, err
	}

	// 2. Validate Files (Count & Size)
	if len(files) == 0 {
		return nil, errors.New("at least 1 image file is required")
	}
	if len(files) > 10 {
		return nil, errors.New("maximum 10 image files allowed")
	}

	for _, file := range files {
		if file.Size > 2*1024*1024 { // 2MB
			return nil, errors.New("file " + file.Filename + " is too large (max 2MB)")
		}
		ext := strings.ToLower(file.Filename[strings.LastIndex(file.Filename, ".")+1:])
		if ext != "jpg" && ext != "jpeg" && ext != "png" {
			return nil, errors.New("file " + file.Filename + " has invalid type (only jpg, jpeg, png allowed)")
		}
	}

	// 3. Upload to Cloudinary & Prepare Media
	cld, err := cloudinary.NewFromParams(u.cfg.CloudinaryCloudName, u.cfg.CloudinaryAPIKey, u.cfg.CloudinaryAPISecret)
	if err != nil {
		u.log.Errorf("Failed to init cloudinary: %v", err)
		return nil, errors.New("internal server error")
	}

	var mediaList []entity.ShowcaseMedia

	for i, file := range files {
		fileContent, err := file.Open()
		if err != nil {
			return nil, err
		}
		
		uploadResult, err := cld.Upload.Upload(ctx, fileContent, uploader.UploadParams{
			Folder: "talas/showcases",
		})
		fileContent.Close() // Close immediately after upload

		if err != nil {
			u.log.Errorf("Failed to upload file %s: %v", file.Filename, err)
			return nil, errors.New("failed to upload image: " + file.Filename)
		}

		mediaList = append(mediaList, entity.ShowcaseMedia{
			URL:      uploadResult.SecureURL,
			Type:     "IMAGE",
			Position: i + 1,
		})
	}

	// 4. Create Entity
	categoryID, _ := uuid.Parse(input.CategoryID)
	slug := strings.ReplaceAll(strings.ToLower(input.Title), " ", "-") + "-" + uuid.New().String()[:8]

	showcase := &entity.Showcase{
		UserID:      userID,
		Title:       input.Title,
		Slug:        slug,
		Description: input.Description,
		CategoryID:  categoryID,
		Status:      entity.StatusPublished,
		Tags:        strings.Split(input.Tags, ","),
		Media:       mediaList,
	}

	// 5. Save to DB
	if err := u.repo.Create(showcase); err != nil {
		u.log.Errorf("Failed to create showcase in db: %v", err)
		return nil, err
	}

	return showcase, nil
}
