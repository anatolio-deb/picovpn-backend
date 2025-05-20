package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"time"

	daemon "github.com/anatolio-deb/picovpnd"
	"github.com/anatolio-deb/picovpnd/common"
	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
	"github.com/sethvargo/go-password/password"
	"github.com/sirupsen/logrus"
	"github.com/tonkeeper/tonapi-go"
	"github.com/xssnick/tonutils-go/address"
	// "github.com/xssnick/tonutils-go/liteclient"
)

// Send any text message to the bot after the bot has been started

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	opts := []bot.Option{
		bot.WithDefaultHandler(defaultHandler),
		bot.WithCallbackQueryDataHandler("button", bot.MatchTypePrefix, buyCallbackHandler),
	}

	b, err := bot.New(os.Getenv("TOKEN"), opts...)
	if nil != err {
		// panics for the sake of simplicity.
		// you should handle this error properly in your code.
		panic(err)
	}

	b.RegisterHandler(bot.HandlerTypeMessageText, "try", bot.MatchTypeCommand, tryHandler)
	b.RegisterHandler(bot.HandlerTypeMessageText, "buy", bot.MatchTypeCommand, buyHandler)
	b.RegisterHandler(bot.HandlerTypeMessageText, "link", bot.MatchTypeCommand, walletLinkHandler)

	go LockExpiredUsers(b)

	b.Start(ctx)
}

func walletLinkHandler(ctx context.Context, b *bot.Bot, update *models.Update) {
	_, err := UserGetByTelegramID(update.Message.From.ID)
	if err != nil {
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
							Text:      "Something went wrong 😟",
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
								ChatID:    update.Message.Chat.ID,
								Text:      "Send me your TON wallet to link it to your account",
								ParseMode: models.ParseModeMarkdown,
							})
							if err != nil {
								logrus.Error(err)
							}
							// 							_, err := b.SendMessage(ctx, &bot.SendMessageParams{
							// 								ChatID: update.Message.Chat.ID,
							// 								Text: fmt.Sprintf(
							// 									`Free Trial is activated for your account 👀
							// Use Cisco AnyConnect app to connect to the VPN:
							// - <a href="https://play.google.com/store/apps/details?id=com.cisco.anyconnect.vpn.android.avf&amp;hl=en">Google Play</a>
							// - <a href="https://apps.apple.com/ru/app/cisco-secure-client/id1135064690?l=en-GB">AppStore</a>

							// - Server Address: picovpn.ru
							// - Username: %s
							// - Password: %s
							// `, update.Message.From.Username, passwd,
							// 								),
							// 								ParseMode: models.ParseModeHTML,
							// 							})
							// if err != nil {
							// 	logrus.Error(err)
							// }
						}
					}
				}
			}
		}
	}

}

func buyCallbackHandler(ctx context.Context, b *bot.Bot, update *models.Update) {
	// answering callback query first to let Telegram know that we received the callback query,
	// and we're handling it. Otherwise, Telegram might retry sending the update repetitively
	// as it thinks the callback query doesn't reach to our application. learn more by
	// reading the footnote of the https://core.telegram.org/bots/api#callbackquery type.
	b.AnswerCallbackQuery(ctx, &bot.AnswerCallbackQueryParams{
		CallbackQueryID: update.CallbackQuery.ID,
		ShowAlert:       false,
	})

	_, err := UserGetByTelegramID(update.Message.From.ID)
	if err != nil {
		// TODO: useradd
		logrus.Error(err)
	} else {
		client, err := tonapi.NewClient(tonapi.TestnetTonApiURL, tonapi.WithToken(os.Getenv("TON_API_TOKEN")))
		if err != nil {
			log.Fatal(err)
		}
		acc, err := client.GetAccount(ctx, tonapi.GetAccountParams{
			AccountID: update.Message.Text,
		})
		if err != nil {
			logrus.Error(err)
		} else {
			acc.GetBalance()
		}

		resp, err := client.Request(ctx, http.MethodGet, "transfer", map[string][]string{
			"ADDRESS": {os.Getenv("TON_WALLET")},
			"AMOUNT":  {update.CallbackQuery.Data}},
			nil,
		)
		if err != nil {
			logrus.Error(err)
		}
		logrus.Debugln(resp)
	}
}

// b.SendMessage(ctx, &bot.SendMessageParams{
// 	ChatID: update.CallbackQuery.Message.Message.Chat.ID,
// 	Text:   "You selected the button: " + update.CallbackQuery.Data,
// })

func buyHandler(ctx context.Context, b *bot.Bot, update *models.Update) {
	kb := &models.InlineKeyboardMarkup{
		InlineKeyboard: [][]models.InlineKeyboardButton{
			{
				{Text: "Monthly", CallbackData: "9"},
				{Text: "Half-year", CallbackData: "36"},
				{Text: "Yearly", CallbackData: "108"},
			},
		},
	}

	b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID: update.Message.Chat.ID,
		Text: `Available plans:
		💎 1 Month -   9 TON
		💎 6 Month -  36 TON 
		💎 1 Year  - 108 TON`,
		ReplyMarkup: kb,
	})
}

func defaultHandler(ctx context.Context, b *bot.Bot, update *models.Update) {
	addr, err := address.ParseAddr(update.Message.Text)
	if err != nil {
		logrus.Error(err)
	} else {
		user, err := UserGetByTelegramID(update.Message.From.ID)
		if err != nil {
			logrus.Error(err)
		} else {
			user.Wallet = addr.String()
			result := DB.Model(&user).Update("wallet", addr.String())
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
				logrus.Error(result.Error)
				_, err := b.SendMessage(ctx, &bot.SendMessageParams{
					ChatID:    update.Message.Chat.ID,
					Text:      "Wallet is linked ✅",
					ParseMode: models.ParseModeMarkdown,
				})
				if err != nil {
					logrus.Error(result.Error)
				}
			}
		}
	}

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
							Text:      "Something went wrong 😟",
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
