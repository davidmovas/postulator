package images

import "github.com/davidmovas/postulator/internal/kernel/settings"

const DefaultOpenAIModel = "gpt-image-2"

var (
	localDirSetting    = settings.String("images.localDir", "")
	openAIModelSetting = settings.String("images.openaiModel", DefaultOpenAIModel, settings.NonEmpty())
)

func LocalDir(values *settings.Values) string {
	return localDirSetting.Get(values)
}

func OpenAIModel(values *settings.Values) string {
	return openAIModelSetting.Get(values)
}
