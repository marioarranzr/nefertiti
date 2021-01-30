package command

import (
	"fmt"
	"strings"
)

type (
	AboutCommand struct {
		*CommandMeta
	}
)

func (c *AboutCommand) Run(args []string) int {
	text := fmt.Sprintf(`
NAME:
  %s is a command-line trading bot that follows a simple but proven
  strategy: buy the dip, then sell those trades as soon as possible, preferably
  on the same day.

USAGE:
  ./%s command [options]

VERSION:
  %s

AUTHOR:
  Mario Arranz <hello@marioarranz.com>

GLOBAL OPTIONS:
  --help     show help
  --version  print the version

PRICING:
  %s is freeware. But if the bot is making you money, then I would
  appreciate a BTC donation here: 1M4ZAsZGA89P54kAawZk8dKTcytLw33keu. Thanks!

DISCLAIMER:
  Never spend money on crypto that you cannot afford to lose. Use at your own
  risk.`, c.AppName, c.AppName, c.AppVersion, c.AppName)

	fmt.Println(strings.TrimSpace(text))

	return 0
}

func (c *AboutCommand) Help() string {
	return "Usage: ./cryptotrader about"
}

func (c *AboutCommand) Synopsis() string {
	return "About " + c.AppName + "."
}
