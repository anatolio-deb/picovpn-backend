package main

import (
	"context"
	"os"
	"os/signal"
	"time"

	daemon "github.com/anatolio-deb/picovpnd"
	"github.com/anatolio-deb/picovpnd/common"
	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
	"github.com/sirupsen/logrus"
)

func LockExpiredUsers(b *bot.Bot) {
	daemonClient, err := daemon.New(common.ListenAddress)
	if err != nil {
		logrus.Error(err)
	}
	ticker := time.NewTicker(time.Minute)
	done := make(chan bool)
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()
	go func() {
		for {
			select {
			case <-done:
				return
			case <-ticker.C:
				h, m, _ := time.Now().Clock()
				if m == 0 && (h == 9 || h == 15) {
					plans, err := PlansGetExpired()
					if err != nil {
						logrus.Error(err)
					}
					for _, p := range plans {
						logrus.Infof("Locking user %d", p.UserID)
						resp := daemonClient.UserLock(p.User.Name)
						if resp.Code > 0 {
							logrus.Error(resp.Error)
						} else {
							go func() {
								_, err := b.SendMessage(ctx, &bot.SendMessageParams{
									ChatID: p.User.ChatID,
									Text: `Hey 👋
We hope you've enjoyed your time with us!
Your subscrition is over now, but we have a special offer for you.
Use /buy command to get it now 🚀
									`,
									ParseMode: models.ParseModeMarkdown,
								})
								if err != nil {
									logrus.Error(err)
								}
							}()
						}
					}
				}
			}
		}
	}()

	<-ctx.Done()
	stop()
	done <- true
}
