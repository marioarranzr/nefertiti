package command

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/marioarranzr/nefertiti/exchanges"
	"github.com/marioarranzr/nefertiti/flag"
)

type (
	MarketsCommand struct {
		*CommandMeta
	}
)

func (c *MarketsCommand) Run(args []string) int {
	flg := flag.Get("exchange")
	if flg.Exists == false {
		return c.ReturnError(errors.New("missing argument: exchange"))
	}

	exchange := exchanges.New().FindByName(flg.String())
	if exchange == nil {
		return c.ReturnError(fmt.Errorf("exchange %v does not exist", flg))
	}

	markets, err := exchange.GetMarkets(true, flag.Get("sandbox").String() == "Y")
	if err != nil {
		return c.ReturnError(err)
	}

	out, err := json.Marshal(markets)
	if err != nil {
		return c.ReturnError(err)
	}

	fmt.Println(string(out))

	return 0
}

func (c *MarketsCommand) Help() string {
	text := `
Usage: ./cryptotrader markets [options]

The markets command returns a list of available currency pairs for trading.

Options:
  --exchange=[name]
`
	return strings.TrimSpace(text)
}

func (c *MarketsCommand) Synopsis() string {
	return "Get a list of available currency pairs for trading."
}
