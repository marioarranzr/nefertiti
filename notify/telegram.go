package notify

import (
	"errors"
	"strconv"

	"github.com/marioarranzr/nefertiti/flag"
	"github.com/marioarranzr/nefertiti/model"
	"github.com/marioarranzr/nefertiti/passphrase"
	"github.com/yanzay/tbot"
)

type Telegram struct {
	appKey string
	chatId int64
}

func (this *Telegram) PromptForKeys(interactive, verify bool) (ok bool, err error) {
	this.appKey = flag.Get("telegram-app-key").String()

	if this.appKey == "none" {
		return false, nil
	}

	if this.appKey == "" && interactive {
		var data []byte
		if data, err = passphrase.Read("Telegram application key"); err != nil {
			return false, err
		}
		this.appKey = string(data)
	}

	if this.appKey != "" {
		chatId := flag.Get("telegram-chat-id").String()
		if chatId != "" {
			this.chatId, err = strconv.ParseInt(chatId, 10, 64)
		} else {
			if interactive {
				var data []byte
				if data, err = passphrase.Read("Telegram chat ID"); err != nil {
					return false, err
				}
				this.chatId, err = strconv.ParseInt(string(data), 10, 64)
			}
		}
		if err != nil {
			return false, errors.New("error when getting Telegram chat ID")
		}
	}

	return this.appKey != "" && this.chatId != 0, nil
}

func (this *Telegram) SendMessage(message string, title string) error {
	if this.appKey == "" {
		return errors.New("missing argument: Telegram application key")
	}
	if this.chatId == 0 {
		return errors.New("missing argument: Telegram chatId")
	}
	bot, err := tbot.NewServer(this.appKey)
	if err != nil {
		return err
	}
	if title == "" {
		return bot.Send(this.chatId, message)
	} else {
		return bot.Send(this.chatId, (title + ": " + message))
	}
}

func NewTelegram() model.Notify {
	return &Telegram{}
}
