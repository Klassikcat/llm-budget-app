package service

import "errors"

var (
	errPriceCatalogRequired             = errors.New("service requires a price catalog")
	errCatalogSyncSourceRequired        = errors.New("service requires a catalog sync source")
	errOpenRouterActivitySourceRequired = errors.New("service requires an OpenRouter activity source")
	errSettingsStoreRequired            = errors.New("service requires a settings store")
	errSecretStoreRequired              = errors.New("service requires a secret store")
	errIngestionServiceRequired         = errors.New("service requires an ingestion service")
	errUsageEntryRepositoryRequired     = errors.New("service requires a usage entry repository")
	errSessionRepositoryRequired        = errors.New("service requires a session repository")
	errSubscriptionRepoRequired         = errors.New("service requires a subscription repository")
	errSubscriptionIDRequired           = errors.New("service requires a subscription id")
	errBudgetRepositoryRequired         = errors.New("service requires a budget repository")
	errForecastRepositoryRequired       = errors.New("service requires a forecast repository")
	errInsightRepositoryRequired        = errors.New("service requires an insight repository")
	errAlertRepositoryRequired          = errors.New("service requires an alert repository")
	errCheckpointRepositoryRequired     = errors.New("service requires a checkpoint repository")
	errFileWatcherRequired              = errors.New("service requires a file watcher")
	errSessionIDRequired                = errors.New("service requires a session id")
	errEntryIDRequired                  = errors.New("service requires an entry id")
)
