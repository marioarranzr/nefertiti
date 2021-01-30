package command

import (
	"errors"
	"fmt"
	"strings"

	"github.com/marioarranzr/nefertiti/exchanges"
	"github.com/marioarranzr/nefertiti/flag"
	"github.com/marioarranzr/nefertiti/model"
)

type (
	BaseCommand struct {
		*CommandMeta
	}
)

func (c *BaseCommand) Run(args []string) int {
	var (
		err error
		flg *flag.Flag
	)

	sandbox := false
	flg = flag.Get("sandbox")
	if flg.Exists {
		sandbox = flg.String() == "Y"
	}

	flg = flag.Get("exchange")
	if flg.Exists == false {
		return c.ReturnError(errors.New("missing argument: exchange"))
	}
	exchange := exchanges.New().FindByName(flg.String())
	if exchange == nil {
		return c.ReturnError(fmt.Errorf("exchange %v does not exist", flg))
	}

	var markets []model.Market
	if markets, err = exchange.GetMarkets(true, sandbox); err != nil {
		return c.ReturnError(err)
	}

	flg = flag.Get("market")
	if flg.Exists == false {
		return c.ReturnError(errors.New("missing argument: market"))
	}
	market := flg.String()
	if model.HasMarket(markets, market) == false {
		return c.ReturnError(fmt.Errorf("market %s does not exist", market))
	}

	var out string
	if out, err = model.GetBaseCurr(markets, market); err != nil {
		return c.ReturnError(err)
	}

	fmt.Println(strings.ToUpper(out))

	return 0
}

func (c *BaseCommand) Help() string {
	text := `
Usage: ./cryptotrader base [options]

The base command returns the base symbol for a given market pair.

Options:
  --exchange = name
  --market   = a valid market pair
`
	return strings.TrimSpace(text)
}

func (c *BaseCommand) Synopsis() string {
	return "Get the base symbol for a given market pair."
}
