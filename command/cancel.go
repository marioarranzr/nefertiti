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
	CancelCommand struct {
		*CommandMeta
	}
)

func (c *CancelCommand) Run(args []string) int {
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

	flg = flag.Get("side")
	if flg.Exists == false {
		return c.ReturnError(errors.New("missing argument: side"))
	}
	side := model.NewOrderSide(flg.String())
	if side == model.ORDER_SIDE_NONE {
		return c.ReturnError(fmt.Errorf("side %v is invalid", flg))
	}

	var client interface{}
	if client, err = exchange.GetClient(true, sandbox); err != nil {
		return c.ReturnError(err)
	}

	if err = exchange.Cancel(client, market, side); err != nil {
		return c.ReturnError(err)
	}

	return 0
}

func (c *CancelCommand) Help() string {
	text := `
Usage: ./cryptotrader cancel [options]

The cancel command cancels all your buy or sell orders on a given market.

Options:
  --exchange = name, for example: Binance (optional, by default Binance)
  --market   = a valid market pair
  --side     = [buy|sell]
`
	return strings.TrimSpace(text)
}

func (c *CancelCommand) Synopsis() string {
	return "Cancels all your buy or sell orders on a given market."
}
