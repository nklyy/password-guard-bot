package main

import (
	"errors"
	"log"
	"password-manager-bot/config"
	"password-manager-bot/internal/bot"
	"password-manager-bot/pkg/logger"
	"password-manager-bot/pkg/mongodb"
	"syscall"

	"go.uber.org/zap"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

func main() {
	cfg, err := config.GetConfig()
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	// Init logger
	newLogger, err := logger.NewLogger(cfg.Environment)
	if err != nil {
		log.Fatalf("can't create logger: %v", err)
	}

	zapLogger, err := newLogger.SetupZapLogger()
	if err != nil {
		log.Fatalf("can't setup zap logger: %v", err)
	}
	defer func(zapLogger *zap.SugaredLogger) {
		err := zapLogger.Sync()
		if err != nil && !errors.Is(err, syscall.ENOTTY) {
			log.Fatalf("can't setup zap logger: %v", err)
		}
	}(zapLogger)

	// Connect to database
	db, ctx, cancel, err := mongodb.NewConnection(cfg)
	if err != nil {
		zapLogger.Fatalf("failed to connect to mongodb: %v", err)
	}
	defer mongodb.Close(db, ctx, cancel)

	// Ping db
	err = mongodb.Ping(db, ctx)
	if err != nil {
		log.Fatal(err)
	}
	zapLogger.Info("DB connected successfully")

	// Repositories
	botRepository, err := bot.NewRepository(db, cfg.MongoDbName, zapLogger)
	if err != nil {
		zapLogger.Fatalf("failed to create bot repository: %v", err)
	}

	botApi, err := tgbotapi.NewBotAPI(cfg.TELEGRAM_KEY)
	if err != nil {
		log.Panic(err)
	}

	botClient, err := bot.NewClient(botApi, botRepository, zapLogger)
	if err != nil {
		zapLogger.Fatalf("failed to create bot service: %v", err)
	}

	botClient.StartBot()
}
