package config_test

import (
	"os"
	"password-guard-bot/config"
	"reflect"
	"testing"
)

func TestInit(t *testing.T) {
	type env struct {
		environment string
		telegramKey string
		mongoDbName string
		mongoDbUrl  string
		iteration   string
	}

	type args struct {
		env env
	}

	setEnv := func(env env) {
		os.Setenv("APP_ENV", env.environment)
		os.Setenv("TELEGRAM_KEY", env.telegramKey)
		os.Setenv("MONGO_DB_NAME", env.mongoDbName)
		os.Setenv("MONGO_DB_URL", env.mongoDbUrl)
		os.Setenv("ITERATION", env.iteration)
	}

	tests := []struct {
		name      string
		args      args
		want      *config.Config
		wantError bool
	}{
		{
			name: "Test config file!",
			args: args{
				env: env{
					environment: "development",
					telegramKey: "example",
					mongoDbName: "example",
					mongoDbUrl:  "http://127.0.0.1",
					iteration:   "1234",
				},
			},
			want: &config.Config{
				Environment: "development",
				TelegramKey: "example",
				MongoDb: config.MongoDb{
					MongoDbName: "example",
					MongoDbUrl:  "http://127.0.0.1",
				},
				Crypto: config.Crypto{
					Iteration: 1234,
				},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			setEnv(test.args.env)

			got, err := config.GetConfig()
			if (err != nil) != test.wantError {
				t.Errorf("Init() error = %v, wantErr %v", err, test.wantError)

				return
			}

			if !reflect.DeepEqual(got, test.want) {
				t.Errorf("Init() got = %v, want %v", got, test.want)
			}
		})
	}
}
