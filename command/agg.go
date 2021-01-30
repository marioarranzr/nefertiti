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
	AggCommand struct {
		*CommandMeta
	}
)

func (c *AggCommand) Run(args []string) int {
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

	var dip float64 = 5
	flg = flag.Get("dip")
	if flg.Exists {
		if dip, err = flg.Float64(); err != nil {
			return c.ReturnError(fmt.Errorf("dip %v is invalid", flg))
		}
	}

	var max float64 = 0
	flg = flag.Get("max")
	if flg.Exists {
		if max, err = flg.Float64(); err != nil {
			return c.ReturnError(fmt.Errorf("max %v is invalid", flg))
		}
	}

	var min float64 = 0
	flg = flag.Get("min")
	if flg.Exists {
		if min, err = flg.Float64(); err != nil {
			return c.ReturnError(fmt.Errorf("min %v is invalid", flg))
		}
	}

	var agg float64
	if agg, err = model.GetAgg(exchange, market, dip, max, min, 4, sandbox); err != nil {
		return c.ReturnError(err)
	}

	fmt.Println(agg)

	return 0
}

func (c *AggCommand) Help() string {
	text := `
Usage: ./cryptotrader agg [options]

The agg command calculates the aggregation level for a given market pair.

Options:
  --exchange = name
  --market   = a valid market pair
  --dip      = percentage that will kick the bot into action (optional, defaults to 5%)
  --max      = maximum price that you will want to pay for the coins (optional)
  --min      = minimum price (optional, defaults to a value 33% below ticker price)
`
	return strings.TrimSpace(text)
}

func (c *AggCommand) Synopsis() string {
	return "Calculates the aggregation level for a given market pair."
}
