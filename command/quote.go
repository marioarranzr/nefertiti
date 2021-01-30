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
	QuoteCommand struct {
		*CommandMeta
	}
)

func (c *QuoteCommand) Run(args []string) int {
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
	exchangeName := ""
	if flg.Exists == false {
		exchangeName = defaultExchange
		// 	return c.ReturnError(errors.New("missing argument: exchange"))
	}
	if exchangeName == "" {
		exchangeName = flg.String()
	}
	exchange := exchanges.New().FindByName(exchangeName)
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
	if out, err = model.GetQuoteCurr(markets, market); err != nil {
		return c.ReturnError(err)
	}

	fmt.Println(strings.ToUpper(out))

	return 0
}

func (c *QuoteCommand) Help() string {
	text := `
Usage: ./cryptotrader quote [options]

The quote command returns the quote symbol for a given market pair.

Options:
  --exchange = name, for example: Binance (optional, by default Binance), for example: Binance (optional, by default Binance)
  --market   = a valid market pair
`
	return strings.TrimSpace(text)
}

func (c *QuoteCommand) Synopsis() string {
	return "Get the quote symbol for a given market pair."
}
