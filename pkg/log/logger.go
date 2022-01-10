package log

import (
	"sync"

	"go.uber.org/zap"
)

var (
	sugarLogger *zap.SugaredLogger
	once        sync.Once
)

// todo: add flags to customize logger
func InitLogger(dev bool) (err error) {
	once.Do(func() {
		var logger *zap.Logger
		if dev {
			logger, err = zap.NewDevelopment()
		} else {
			logger, err = zap.NewProduction()
		}

		if err != nil {
			return
		}
		sugarLogger = logger.Sugar()
	})
	return err
}

// GetLogger return default logger
func GetLogger() *zap.SugaredLogger {
	return sugarLogger
}
