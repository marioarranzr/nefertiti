package command

import (
	"errors"
	"strings"

	"github.com/marioarranzr/nefertiti/flag"
	"github.com/marioarranzr/nefertiti/model"
	"github.com/marioarranzr/nefertiti/notify"
)

type (
	NotifyCommand struct {
		*CommandMeta
	}
)

func (c *NotifyCommand) Run(args []string) int {
	var err error

	var service model.Notify = nil
	if service, err = notify.New().Init(flag.Interactive(), true); err != nil {
		return c.ReturnError(err)
	}

	if service == nil {
		return c.ReturnError(errors.New("Service not found or not initialized. Quitting."))
	}

	if err = service.SendMessage(flag.Get("msg").String(), flag.Get("title").String()); err != nil {
		return c.ReturnError(err)
	}

	return 0
}

func (c *NotifyCommand) Help() string {
	text := `
Usage: ./cryptotrader notify [options]

The notify command sends a notification.

Options:
  --pushover-app-key=X
  --pushover-user-key=Y

or:
  --telegram-app-key=X
  --telegram-chat-id=Y

and:
  --msg="blah blah blah"
`
	return strings.TrimSpace(text)
}

func (c *NotifyCommand) Synopsis() string {
	return "Send a notification."
}
