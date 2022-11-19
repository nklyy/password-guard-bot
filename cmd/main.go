package main

import (
	"context"
	"errors"
	"log"
	"password-guard-bot/config"
	"password-guard-bot/internal/bot"
	"password-guard-bot/pkg/crypto"
	"password-guard-bot/pkg/logger"
	"password-guard-bot/pkg/mongodb"
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

	cryptoService, err := crypto.NewCryptoService()
	if err != nil {
		zapLogger.Fatalf("failed to create crypto service: %v", err)
	}

	// Repositories
	botRepository, err := bot.NewRepository(db, cfg.MongoDbName, zapLogger)
	if err != nil {
		zapLogger.Fatalf("failed to create bot repository: %v", err)
	}

	err = botRepository.CreateUniqueIndexes(context.Background())
	if err != nil {
		zapLogger.Fatalf("failed to create bot repository indexes: %v", err)
	}

	botApi, err := tgbotapi.NewBotAPI(cfg.TELEGRAM_KEY)
	if err != nil {
		log.Panic(err)
	}
	// botApi.Debug = true

	messageService, err := bot.NewMessageService(botApi, zapLogger)
	if err != nil {
		zapLogger.Fatalf("failed to create message service: %v", err)
	}

	botService, err := bot.NewService(botApi, cryptoService, botRepository, zapLogger)
	if err != nil {
		zapLogger.Fatalf("failed to create bot service: %v", err)
	}

	botClient, err := bot.NewClient(botService, messageService, zapLogger)
	if err != nil {
		zapLogger.Fatalf("failed to create bot service: %v", err)
	}

	updateConfig := tgbotapi.NewUpdate(0)
	updateConfig.Timeout = 60

	updates := botApi.GetUpdatesChan(updateConfig)
	botClient.StartBot(updates)
}
