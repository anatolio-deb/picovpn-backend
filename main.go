package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"

	daemon "github.com/anatolio-deb/picovpnd"
	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
	"github.com/sethvargo/go-password/password"
	"github.com/sirupsen/logrus"
)

// Send any text message to the bot after the bot has been started

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	opts := []bot.Option{
		bot.WithDefaultHandler(defaultHandler),
	}

	b, err := bot.New(os.Getenv("TOKEN"), opts...)
	if nil != err {
		// panics for the sake of simplicity.
		// you should handle this error properly in your code.
		panic(err)
	}

	b.RegisterHandler(bot.HandlerTypeMessageText, "start", bot.MatchTypeCommand, startHandler)
	b.RegisterHandlerMatchFunc(matchFunc, helloHandler)

	b.Start(ctx)
}

func matchFunc(update *models.Update) bool {
	if update.Message == nil {
		return false
	}
	return update.Message.Text == "hello"
}

func helloHandler(ctx context.Context, b *bot.Bot, update *models.Update) {
	log.Printf("hello handler")
}

func defaultHandler(ctx context.Context, b *bot.Bot, update *models.Update) {
	log.Printf("default handler")
}

func startHandler(ctx context.Context, b *bot.Bot, update *models.Update) {
	_, err := UserGetByTelegramID(update.Message.From.ID)
	if err == nil {
		_, err := b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID:    update.Message.Chat.ID,
			Text:      "Multiple accounts are not allowed",
			ParseMode: models.ParseModeMarkdown,
		})
		if err != nil {
			logrus.Error(err)
		}
	} else {
		logrus.Error(err)
		passwd, err := password.Generate(8, 4, 0, true, true)
		if err != nil {
			logrus.Error(err)
			_, err := b.SendMessage(ctx, &bot.SendMessageParams{
				ChatID:    update.Message.Chat.ID,
				Text:      "Что-то пошло не так...",
				ParseMode: models.ParseModeMarkdown,
			})
			if err != nil {
				logrus.Error(err)
			}
		}
		response := daemon.UserAdd(update.Message.From.Username, passwd)
		if response.Code > 0 {
			logrus.Error(response.Error)
			_, err := b.SendMessage(ctx, &bot.SendMessageParams{
				ChatID:    update.Message.Chat.ID,
				Text:      "Что-то пошло не так...",
				ParseMode: models.ParseModeMarkdown,
			})
			if err != nil {
				logrus.Error(err)
			}
		} else {
			plan := UserPlan{Type: Monthly}
			result := DB.Create(&plan)
			if result.Error != nil {
				logrus.Error(result.Error)
				_, err := b.SendMessage(ctx, &bot.SendMessageParams{
					ChatID:    update.Message.Chat.ID,
					Text:      "Что-то пошло не так...",
					ParseMode: models.ParseModeMarkdown,
				})
				if err != nil {
					logrus.Error(result.Error)
				}
			}
			user := &User{
				PlanID:     plan.ID,
				Plan:       plan,
				TelegramID: update.Message.From.ID,
			}
			result = DB.Create(&user)
			if result.Error != nil {
				logrus.Error(result.Error)
				_, err := b.SendMessage(ctx, &bot.SendMessageParams{
					ChatID:    update.Message.Chat.ID,
					Text:      "Что-то пошло не так...",
					ParseMode: models.ParseModeMarkdown,
				})
				if err != nil {
					logrus.Error(err)
				}
			} else {
				logrus.Infof("created new user ID %d", user.ID)
				_, err := b.SendMessage(ctx, &bot.SendMessageParams{
					ChatID: update.Message.Chat.ID,
					Text: fmt.Sprintf(
						`Free Trial is activated for your account.
							Use Cisco AnyConnect apps to connect to the VPN:
							1. Google Play: https://play.google.com/store/apps/details?id=com.cisco.anyconnect.vpn.android.avf&hl=en
							2. AppStore: https://apps.apple.com/ru/app/cisco-secure-client/id1135064690?l=en-GB

							- Server Address: picovpn.ru
							- Username: %s
							- Password: ||%s||
							`, update.Message.From.Username, update.Message.Text,
					),
					ParseMode: models.ParseModeMarkdown,
				})
				if err != nil {
					logrus.Error(err)
				}
			}
		}
	}

}
