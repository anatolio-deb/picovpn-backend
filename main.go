package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"

	picoDaemonClient "github.com/anatolio-deb/picovpnd"
	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
	"github.com/sirupsen/logrus"
)

// Send any text message to the bot after the bot has been started

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	opts := []bot.Option{
		// bot.WithDefaultHandler(defaultHandler),
	}

	b, err := bot.New(os.Getenv("TOKEN"), opts...)
	if nil != err {
		// panics for the sake of simplicity.
		// you should handle this error properly in your code.
		panic(err)
	}

	b.RegisterHandler(bot.HandlerTypeMessageText, "signup", bot.MatchTypeCommand, signupHandler)
	b.RegisterHandler(bot.HandlerTypeMessageText, "signin", bot.MatchTypeCommandStartOnly, signinHandler)

	b.Start(ctx)
}

func signupHandler(ctx context.Context, b *bot.Bot, update *models.Update) {
	user := UserGetByTelegramID(update.Message.From.ID)
	if user != nil {
		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: update.Message.Chat.ID,
			Text: fmt.Sprintf(`Мы обнаружили, что вам пренадлежит аккаунт %s. 
			Вы можете войти в него, так как множество аккаунтов не поддерживается нашим сервисом.`, update.Message.From.Username),
			ParseMode: models.ParseModeMarkdown,
		})
		return
	}
	if update.Message != nil {
		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID:    update.Message.Chat.ID,
			Text:      "Придумай и отправь мне новый пароль",
			ParseMode: models.ParseModeMarkdown,
		})
	}
	attempts := 100
	for attempts > 0 {
		if update.Message != nil {
			response := picoDaemonClient.UserAdd(update.Message.From.Username, update.Message.Text)
			if response.Code > 0 {
				logrus.Debug(response.Error)
				b.SendMessage(ctx, &bot.SendMessageParams{
					ChatID:    update.Message.Chat.ID,
					Text:      "Что-то пошло не так...",
					ParseMode: models.ParseModeMarkdown,
				})
			} else {
				plan := UserPlan{Type: Monthly}
				result := DB.Create(&plan)
				if plan.ID == 0 {
					logrus.Error(result.Error)
					b.SendMessage(ctx, &bot.SendMessageParams{
						ChatID:    update.Message.Chat.ID,
						Text:      "Что-то пошло не так...",
						ParseMode: models.ParseModeMarkdown,
					})
				} else {
					logrus.Debugf("created new user ID %d", plan.ID)
				}
				user = &User{
					PlanID:     plan.ID,
					Plan:       plan,
					TelegramID: update.Message.From.ID,
				}
				result = DB.Create(&user)
				if user.ID == 0 {
					logrus.Error(result.Error)
					b.SendMessage(ctx, &bot.SendMessageParams{
						ChatID:    update.Message.Chat.ID,
						Text:      "Что-то пошло не так...",
						ParseMode: models.ParseModeMarkdown,
					})
				} else {
					logrus.Debugf("created new user ID %d", user.ID)
				}

				b.SendMessage(ctx, &bot.SendMessageParams{
					ChatID: update.Message.Chat.ID,
					Text: fmt.Sprintf(
						`Привет, подключили тебе месяц триала!
							Для подключения используй Cisco AnyConnect:
							1. Google Play: https://play.google.com/store/apps/details?id=com.cisco.anyconnect.vpn.android.avf&hl=en
							2. AppStore: https://apps.apple.com/ru/app/cisco-secure-client/id1135064690?l=en-GB
	
							- Адрес сервера: picovpn.ru
							- Имя пользователя: %s
							- Пароль: ||%s||
	
							Приятного серфинга без рекламы 👀
							`, update.Message.From.Username, update.Message.Text,
					),
					ParseMode: models.ParseModeMarkdown,
				})
			}
			attempts = 0
		} else {
			attempts--
		}
	}
}

func signinHandler(ctx context.Context, b *bot.Bot, update *models.Update) {
	b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID:    update.Message.Chat.ID,
		Text:      "Caught *bar*",
		ParseMode: models.ParseModeMarkdown,
	})
}

// func defaultHandler(ctx context.Context, b *bot.Bot, update *models.Update) {
// 	b.SendMessage(ctx, &bot.SendMessageParams{
// 		ChatID:    update.Message.Chat.ID,
// 		Text:      "Say message with `/foo` anywhere or with `/bar` at start of the message",
// 		ParseMode: models.ParseModeMarkdown,
// 	})
// }
