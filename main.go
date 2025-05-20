package main

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"time"

	daemon "github.com/anatolio-deb/picovpnd"
	"github.com/anatolio-deb/picovpnd/common"
	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
	"github.com/sethvargo/go-password/password"
	"github.com/sirupsen/logrus"
	"github.com/tonkeeper/tonapi-go"
	"github.com/xssnick/tonutils-go/address"
	"github.com/xssnick/tonutils-go/liteclient"
	"github.com/xssnick/tonutils-go/tlb"
	"github.com/xssnick/tonutils-go/ton"
	"github.com/xssnick/tonutils-go/ton/wallet"
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
						}
					}
				}
			}
		}
	}
	_, err = b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID:    update.Message.Chat.ID,
		Text:      "Send me your TON wallet to link it to your account",
		ParseMode: models.ParseModeMarkdown,
	})
	if err != nil {
		logrus.Error(err)
	}
}

func buyCallbackHandler(ctx context.Context, b *bot.Bot, update *models.Update) {
	// answering callback query first to let Telegram know that we received the callback query,
	// and we're handling it. Otherwise, Telegram might retry sending the update repetitively
	// as it thinks the callback query doesn't reach to our application. learn more by
	// reading the footnote of the https://core.telegram.org/bots/api#callbackquery type.
	_, err := b.AnswerCallbackQuery(ctx, &bot.AnswerCallbackQueryParams{
		CallbackQueryID: update.CallbackQuery.ID,
		ShowAlert:       false,
	})
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
	} else {

		callbackData := strings.Split(update.CallbackQuery.Data, ";")
		if len(callbackData) < 1 {
			logrus.Error("invalid callback data")
			_, err := b.SendMessage(ctx, &bot.SendMessageParams{
				ChatID:    update.Message.Chat.ID,
				Text:      "Something went wrong 😟",
				ParseMode: models.ParseModeMarkdown,
			})
			if err != nil {
				logrus.Error(err)
			}
		} else {
			// useradd or get existing; set new expire date from amount
			userID, err := strconv.Atoi(callbackData[1])
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
			user, err := UserGetByTelegramID(int64(userID))
			if err != nil {
				logrus.Error(err)
				_, err := b.SendMessage(ctx, &bot.SendMessageParams{
					ChatID:    update.Message.Chat.ID,
					Text:      "Use /link to connect your TON wallet",
					ParseMode: models.ParseModeMarkdown,
				})
				if err != nil {
					logrus.Error(err)
				}
			} else {
				// initialize connection pool.
				testnetConfigURL := "https://ton-blockchain.github.io/testnet-global.config.json"
				conn := liteclient.NewConnectionPool()
				ctx := context.Background()
				err := conn.AddConnectionsFromConfigUrl(ctx, testnetConfigURL)
				if err != nil {
					logrus.Error(err)
				} else {
					// initialize api client.
					api := ton.NewAPIClient(conn)

					// // importing wallet.
					seedStr := user.Wallet // if you don't have one you can generate it with tonwallet.NewSeed().
					seed := strings.Split(seedStr, " ")

					w, err := wallet.FromSeed(api, seed, wallet.V4R2)
					if err != nil {
						logrus.Error(err)
					} else {
						log.Println("WALLET ADDRESS: ", w.Address().String())

						// getting latest master chain.
						block, err := api.CurrentMasterchainInfo(ctx)
						if err != nil {
							logrus.Error(err)
						}

						balance, err := w.GetBalance(ctx, block)
						if err != nil {
							logrus.Error(err)
						} else {
							log.Println("AVAILABLE BALANCE", balance)

							var amount uint64
							switch callbackData[0] {
							case "button_1":
								amount = TON * 3
							case "button_2":
								amount = TON * 3
							case "button_3":
								amount = TON * 108
							}

							// check if we have enough balance.
							if balance.Nano().Uint64() < amount {
								logrus.Error(errors.New("insufficient balance"))
							} else {
								// parse address, in case we receive an invalid address.
								addr, err := address.ParseAddr(os.Getenv("TON_WALLET"))
								if err != nil {
									logrus.Error(err)
								} else {
									// Now we can use the method Transfer that the library provides.
									// Which absolutely fine, the problem is that we WANT to retrieve the hash of the transaction.
									// Currently the Transfer method doesn't not return the hash of the transaction, because it gives you
									// the option to not wait for the transaction to finish. This is my assumption of course.
									// So let's try to wait for the transaction and to retrieve the hash of the transaction.
									// For that purpose the library provides us with a method called SendManyWaitTxHash.

									// creating cell for comment.
									// body, err := tonwallet.CreateCommentCell(comment)
									// if err != nil {
									// 	panic(err)
									// }

									txn, err := w.SendManyWaitTxHash(ctx, []*wallet.Message{
										{
											Mode: 1,
											InternalMessage: &tlb.InternalMessage{
												IHRDisabled: true,
												Bounce:      false, // we don't want the transaction to bounce, but you can change it to true if you want.
												DstAddr:     addr,  // destination address.
												Amount:      tlb.FromNanoTONU(amount),
												Body:        nil,
											},
										},
									})
									if err != nil {
										logrus.Error(err)
									} else {
										// now we can use this transaction hash to search
										// the transaction in tonscan explorer.
										txnHash := base64.StdEncoding.EncodeToString(txn)
										logrus.Info("TXN HASH: ", txnHash)
									}
								}
							}
						}
					}
				}
			}
		}
	}
}

func buyHandler(ctx context.Context, b *bot.Bot, update *models.Update) {
	kb := &models.InlineKeyboardMarkup{
		InlineKeyboard: [][]models.InlineKeyboardButton{
			{
				{Text: "Monthly", CallbackData: fmt.Sprintf("button_1;%d", update.Message.From.ID)},
				{Text: "Half-year", CallbackData: fmt.Sprintf("button_1;%d", update.Message.From.ID)},
				{Text: "Yearly", CallbackData: fmt.Sprintf("button_1;%d", update.Message.From.ID)},
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
	client, err := tonapi.NewClient(tonapi.TestnetTonApiURL, tonapi.WithToken(os.Getenv("TON_API_TOKEN")))
	if err != nil {
		logrus.Error(err)
	} else {
		addr, err := client.AddressParse(ctx, tonapi.AddressParseParams{
			AccountID: update.Message.Text,
		})
		if err != nil {
			logrus.Error(err)
			_, err := b.SendMessage(ctx, &bot.SendMessageParams{
				ChatID:    update.Message.Chat.ID,
				Text:      "Please, send me a vaild TON wallet 😟",
				ParseMode: models.ParseModeMarkdown,
			})
			if err != nil {
				logrus.Error(err)
			}
		} else {
			user, err := UserGetByTelegramID(update.Message.From.ID)
			if err != nil {
				logrus.Error(err)
			} else {
				result := DB.Model(&user).Update("wallet", addr.GetRawForm())
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

}

func tryHandler(ctx context.Context, b *bot.Bot, update *models.Update) {
	_, err := UserGetByTelegramID(update.Message.From.ID)
	if err != nil {
		logrus.Error(err)
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
	} else {
		_, err := b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID:    update.Message.Chat.ID,
			Text:      "Multiple accounts are not allowed 🙇",
			ParseMode: models.ParseModeMarkdown,
		})
		if err != nil {
			logrus.Error(err)
		}

	}
}
