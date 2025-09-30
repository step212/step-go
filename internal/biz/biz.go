package biz

import "github.com/google/wire"

// ProviderSet is biz providers.
var ProviderSet = wire.NewSet(
	NewGreeterUsecase,
	NewStepUsecase,
	NewMinioUsecase,
	NewStepNoauthUsecase,
	NewAsynqStatisticsUsecase,
	NewAsynqFeedbackUsecase,
	NewPortraitUsecase,
	NewFeedbackUsecase,
)
