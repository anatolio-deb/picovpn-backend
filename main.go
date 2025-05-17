package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"time"

	daemon "github.com/anatolio-deb/picovpnd"
	"github.com/anatolio-deb/picovpnd/common"
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

	b.RegisterHandler(bot.HandlerTypeMessageText, "try", bot.MatchTypeCommand, tryHandler)
	b.RegisterHandler(bot.HandlerTypeMessageText, "buy", bot.MatchTypeCommand, buyHandler)
	// b.RegisterHandlerMatchFunc(matchFunc, helloHandler)
	go LockExpiredUsers(b)

	b.Start(ctx)
}

// func matchFunc(update *models.Update) bool {
// 	if update.Message == nil {
// 		return false
// 	}
// 	return update.Message.Text == "hello"
// }

func buyHandler(ctx context.Context, b *bot.Bot, update *models.Update) {

}

func passwordHandler(ctx context.Context, b *bot.Bot, update *models.Update) {

}

func defaultHandler(ctx context.Context, b *bot.Bot, update *models.Update) {
	log.Printf("default handler")
}

func tryHandler(ctx context.Context, b *bot.Bot, update *models.Update) {
	_, err := UserGetByTelegramID(update.Message.From.ID)
	if err != nil {
		logrus.Error(err)
	} else {
		_, err := b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID:    update.Message.Chat.ID,
			Text:      "Multiple accounts are not allowed 🙇",
			ParseMode: models.ParseModeMarkdown,
		})
		if err != nil {
			logrus.Error(err)
		} else {
			passwd, err := password.Generate(8, 4, 0, true, true)
			if err != nil {
				logrus.Error(err)
				_, err := b.SendMessage(ctx, &bot.SendMessageParams{
					ChatID:    update.Message.Chat.ID,
					Text:      "Something went wrong 😟",
					ParseMode: models.ParseModeMarkdown,
				})
				if err != nil {
					logrus.Error(err)
				}
			}
			daemonClient, err := daemon.New(common.ListenAddress)
			if err != nil {
				logrus.Error(err)
			} else {
				response := daemonClient.UserAdd(update.Message.From.Username, passwd)
				logrus.Error(response.Code, response.Error)
				if response.Code > 0 {
					logrus.Error(response.Error)
					_, err := b.SendMessage(ctx, &bot.SendMessageParams{
						ChatID:    update.Message.Chat.ID,
						Text:      "Something went wrong 😟",
						ParseMode: models.ParseModeMarkdown,
					})
					if err != nil {
						logrus.Error(err)
					}
				} else {
					user := User{
						// PlanID:     plan.ID,
						// Plan:       plan,
						TelegramID: update.Message.From.ID,
						ChatID:     update.Message.Chat.ID,
						Name:       update.Message.From.Username,
					}
					result := DB.Create(&user)
					if result.Error != nil {
						logrus.Error(result.Error)
						_, err := b.SendMessage(ctx, &bot.SendMessageParams{
							ChatID:    update.Message.Chat.ID,
							Text:      "Something went wrong",
							ParseMode: models.ParseModeMarkdown,
						})
						if err != nil {
							logrus.Error(err)
						}
					} else {
						plan := UserPlan{
							Type:      Monthly,
							ExpiresAt: time.Now().AddDate(0, 1, 0),
							UserID:    user.ID,
							User:      user,
						}
						result = DB.Create(&plan)
						if result.Error != nil {
							logrus.Error(result.Error)
							_, err := b.SendMessage(ctx, &bot.SendMessageParams{
								ChatID:    update.Message.Chat.ID,
								Text:      "Something went wrong 😟",
								ParseMode: models.ParseModeMarkdown,
							})
							if err != nil {
								logrus.Error(result.Error)
							}
						} else {
							logrus.Infof("created new user ID %d", user.ID)
							_, err := b.SendMessage(ctx, &bot.SendMessageParams{
								ChatID: update.Message.Chat.ID,
								Text: fmt.Sprintf(
									`Free Trial is activated for your account 👀
Use Cisco AnyConnect app to connect to the VPN:
- <a href="https://play.google.com/store/apps/details?id=com.cisco.anyconnect.vpn.android.avf&amp;hl=en">Google Play</a>
- <a href="https://apps.apple.com/ru/app/cisco-secure-client/id1135064690?l=en-GB">AppStore</a>

- Server Address: picovpn.ru
- Username: %s
- Password: %s
`, update.Message.From.Username, passwd,
								),
								ParseMode: models.ParseModeHTML,
							})
							if err != nil {
								logrus.Error(err)
							}
						}
					}
				}
			}
		}

	}
}
