package sitemap

import "github.com/davidmovas/postulator/internal/domain/entities"

const (
	WPStatusPublish = "publish"
	WPStatusDraft   = "draft"
	WPStatusPending = "pending"
	WPStatusPrivate = "private"
)

func ArticleStatusToWPStatus(status entities.ArticleStatus) string {
	switch status {
	case entities.StatusPublished:
		return WPStatusPublish
	case entities.StatusDraft:
		return WPStatusDraft
	case entities.StatusPending:
		return WPStatusPending
	case entities.StatusPrivate:
		return WPStatusPrivate
	default:
		return WPStatusDraft
	}
}

func WPStatusToArticleStatus(wpStatus string) entities.ArticleStatus {
	switch wpStatus {
	case WPStatusPublish:
		return entities.StatusPublished
	case WPStatusDraft:
		return entities.StatusDraft
	case WPStatusPending:
		return entities.StatusPending
	case WPStatusPrivate:
		return entities.StatusPrivate
	default:
		return entities.StatusDraft
	}
}

func WPStatusToPublishStatus(wpStatus string) entities.NodePublishStatus {
	switch wpStatus {
	case WPStatusPublish:
		return entities.PubStatusPublished
	case WPStatusDraft:
		return entities.PubStatusDraft
	case WPStatusPending:
		return entities.PubStatusPending
	default:
		return entities.PubStatusNone
	}
}

func PublishStatusToWPStatus(status entities.NodePublishStatus) string {
	switch status {
	case entities.PubStatusPublished:
		return WPStatusPublish
	case entities.PubStatusDraft:
		return WPStatusDraft
	case entities.PubStatusPending:
		return WPStatusPending
	default:
		return WPStatusDraft
	}
}
