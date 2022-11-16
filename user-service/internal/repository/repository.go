package repository

import (
	"github.com/google/wire"
	"user-service/internal/model"
)

// ProviderSet is repo providers.
var ProviderSet = wire.NewSet(model.DB, NewUserInfo)
