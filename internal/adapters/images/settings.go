package images

import "github.com/davidmovas/postulator/internal/kernel/settings"

const (
	DefaultOpenAIModel   = "gpt-image-2"
	DefaultOpenAIQuality = "medium"
)

var (
	localDirSetting      = settings.String("images.localDir", "")
	openAIModelSetting   = settings.String("images.openaiModel", DefaultOpenAIModel, settings.NonEmpty())
	openAIQualitySetting = settings.Enum("images.openaiQuality", DefaultOpenAIQuality,
		[]string{"low", "medium", "high", "auto"})
)

func LocalDir(values *settings.Values) string {
	return localDirSetting.Get(values)
}

func OpenAIModel(values *settings.Values) string {
	return openAIModelSetting.Get(values)
}

func OpenAIQuality(values *settings.Values) string {
	return openAIQualitySetting.Get(values)
}
