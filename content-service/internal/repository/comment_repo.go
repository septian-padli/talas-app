package repository

import (
	"encoding/base64"
	"encoding/json"

	"github.com/google/uuid"
	"github.com/septianpadli/talas/content-service/internal/entity"
	"gorm.io/gorm"
)

type CommentRepository interface {
	CreateComment(comment *entity.Comment) error
	GetCommentByID(id uuid.UUID) (*entity.Comment, error)
	GetCommentsByShowcaseID(showcaseID uuid.UUID, limit int, cursor string) ([]entity.Comment, []entity.Comment, *entity.PaginationMeta, error)
	UpdateComment(comment *entity.Comment) error
	DeleteComment(id uuid.UUID) error
	ToggleCommentLike(userID uuid.UUID, commentID uuid.UUID) (bool, int, error)
}

type commentRepository struct {
	db *gorm.DB
}

func NewCommentRepository(db *gorm.DB) CommentRepository {
	return &commentRepository{db: db}
}

func (r *commentRepository) CreateComment(comment *entity.Comment) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(comment).Error; err != nil {
			return err
		}

		// Increment Showcase Comments Count
		if err := tx.Model(&entity.Showcase{}).Where("id = ?", comment.ShowcaseID).UpdateColumn("comments_count", gorm.Expr("comments_count + ?", 1)).Error; err != nil {
			return err
		}

		return nil
	})
}

func (r *commentRepository) GetCommentByID(id uuid.UUID) (*entity.Comment, error) {
	var comment entity.Comment
	if err := r.db.Where("id = ?", id).First(&comment).Error; err != nil {
		return nil, err
	}
	return &comment, nil
}

func (r *commentRepository) GetCommentsByShowcaseID(showcaseID uuid.UUID, limit int, cursor string) ([]entity.Comment, []entity.Comment, *entity.PaginationMeta, error) {
	var parents []entity.Comment
	var replies []entity.Comment
	meta := &entity.PaginationMeta{
		HasNext:    false,
		NextCursor: "",
	}

	query := r.db.Unscoped().Where("showcase_id = ? AND parent_id IS NULL", showcaseID).Order("created_at DESC")

	if cursor != "" {
		decodedCursor, err := base64.StdEncoding.DecodeString(cursor)
		if err == nil {
			var pCursor entity.PaginationCursor
			if err := json.Unmarshal(decodedCursor, &pCursor); err == nil {
				query = query.Where("(created_at < ? OR (created_at = ? AND id < ?))", pCursor.CreatedAt, pCursor.CreatedAt, pCursor.ID)
			}
		}
	}

	// Fetch limit + 1
	if err := query.Limit(limit + 1).Find(&parents).Error; err != nil {
		return nil, nil, nil, err
	}

	if len(parents) > limit {
		meta.HasNext = true
		parents = parents[:limit]
		lastItem := parents[len(parents)-1]

		newCursor := entity.PaginationCursor{
			CreatedAt: lastItem.CreatedAt,
			ID:        lastItem.ID,
		}
		cursorJSON, _ := json.Marshal(newCursor)
		meta.NextCursor = base64.StdEncoding.EncodeToString(cursorJSON)
	}

	if len(parents) == 0 {
		return parents, replies, meta, nil
	}

	// Fetch Replies
	parentIDs := make([]uuid.UUID, len(parents))
	for i, p := range parents {
		parentIDs[i] = p.ID
	}

	if err := r.db.Unscoped().Where("parent_id IN ?", parentIDs).Order("created_at DESC").Find(&replies).Error; err != nil {
		return nil, nil, nil, err
	}

	return parents, replies, meta, nil
}

func (r *commentRepository) UpdateComment(comment *entity.Comment) error {
	return r.db.Save(comment).Error
}

func (r *commentRepository) DeleteComment(id uuid.UUID) error {
	return r.db.Delete(&entity.Comment{}, id).Error
}

// ToggleCommentLike toggles like/unlike on a comment (returns liked status and new count)
func (r *commentRepository) ToggleCommentLike(userID uuid.UUID, commentID uuid.UUID) (bool, int, error) {
	var like entity.CommentLike
	var isLiked bool
	var likesCount int

	err := r.db.Transaction(func(tx *gorm.DB) error {
		// Check if like exists
		result := tx.Where("user_id = ? AND comment_id = ?", userID, commentID).Limit(1).Find(&like)

		if result.Error != nil {
			return result.Error
		}

		if result.RowsAffected > 0 {
			// Found -> Delete (Unlike)
			if err := tx.Delete(&like).Error; err != nil {
				return err
			}
			// Decrement Count
			if err := tx.Model(&entity.Comment{}).Where("id = ?", commentID).UpdateColumn("likes_count", gorm.Expr("likes_count - ?", 1)).Error; err != nil {
				return err
			}
			isLiked = false
		} else {
			// Not Found -> Create (Like)
			newLike := entity.CommentLike{
				UserID:    userID,
				CommentID: commentID,
			}
			if err := tx.Create(&newLike).Error; err != nil {
				return err
			}
			// Increment Count
			if err := tx.Model(&entity.Comment{}).Where("id = ?", commentID).UpdateColumn("likes_count", gorm.Expr("likes_count + ?", 1)).Error; err != nil {
				return err
			}
			isLiked = true
		}

		// Get updated count
		var comment entity.Comment
		if err := tx.Select("likes_count").Where("id = ?", commentID).First(&comment).Error; err != nil {
			return err
		}
		likesCount = comment.LikesCount

		return nil
	})

	return isLiked, likesCount, err
}
