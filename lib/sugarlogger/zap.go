package sugarlogger

import "go.uber.org/zap"

var sugar *zap.SugaredLogger

func GetSugar() *zap.SugaredLogger {
	if sugar == nil {
		logger, _ := zap.NewDevelopment()
		sugar = logger.Sugar()
	}
	return sugar
}
