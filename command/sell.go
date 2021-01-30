package command

import (
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"

	"github.com/marioarranzr/nefertiti/exchanges"
	"github.com/marioarranzr/nefertiti/flag"
	"github.com/marioarranzr/nefertiti/model"
	"github.com/marioarranzr/nefertiti/notify"
	"github.com/marioarranzr/nefertiti/pricing"
)

type (
	SellCommand struct {
		*CommandMeta
	}
)

func (c *SellCommand) Run(args []string) int {
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

	var strategy int64 = int64(model.STRATEGY_STANDARD)
	flg = flag.Get("strategy")
	if flg.Exists == false {
		flag.Set("strategy", strconv.FormatInt(strategy, 10))
	} else {
		if strategy, err = flg.Int64(); err != nil {
			return c.ReturnError(fmt.Errorf("strategy %v is invalid", flg))
		}
	}

	var mult float64 = pricing.FIVE_PERCENT
	flg = flag.Get("mult")
	if flg.Exists == false {
		flag.Set("mult", strconv.FormatFloat(mult, 'f', -1, 64))
	} else {
		if mult, err = flg.Float64(); err != nil {
			return c.ReturnError(fmt.Errorf("mult %v is invalid", flg))
		}
	}

	var all []model.Market
	if all, err = exchange.GetMarkets(true, sandbox); err != nil {
		return c.ReturnError(err)
	}

	hold := flag.Get("hold").String()
	if hold != "" {
		markets := strings.Split(hold, ",")
		for _, market := range markets {
			if model.HasMarket(all, market) == false {
				return c.ReturnError(fmt.Errorf("market %s does not exist", market))
			}
		}
	}

	var level int64 = notify.LEVEL_DEFAULT
	flg = flag.Get("notify")
	if flg.Exists == false {
		flag.Set("notify", strconv.FormatInt(level, 10))
	} else {
		if level, err = flg.Int64(); err != nil {
			return c.ReturnError(fmt.Errorf("notify %v is invalid", flg))
		}
	}

	success := func(service model.Notify) error {
		err := c.ReturnSuccess()
		if err != nil {
			return err
		}
		msg := fmt.Sprintf("Listening to %s...", exchange.GetInfo().Name)
		log.Println("[INFO] " + msg)
		if service != nil {
			if notify.CanSend(notify.Level(), notify.INFO) {
				err := service.SendMessage(msg, fmt.Sprintf("%s - INFO", exchange.GetInfo().Name))
				if err != nil {
					return err
				}
			}
		}
		return nil
	}

	if err = exchange.Sell(time.Now(), strings.Split(hold, ","), sandbox, flag.Exists("tweet"), flag.Debug(), success); err != nil {
		return c.ReturnError(err)
	}

	return 0
}

func (c *SellCommand) Help() string {
	text := `
Usage: ./cryptotrader sell [options]

The sell command listens for buy orders getting filled, and then opens new sell orders for them.

Options:
  --exchange = [name], for example: Binance (optional, by default Binance)
  --sandbox  = [Y|N] (optional)
  --strategy = [0|1|2|3|4] (see below)
  --notify   = [0|1|2|3] (see below)
  --mult     = multiplier, for example: 1.05 (aka 5 percent, optional)
  --hold     = name of the market not to sell, for example: BTC-EUR (optional)

Strategy:
  0 = Standard. No trailing. No stop-loss. Recommended, default strategy.
  1 = Trailing. As strategy #0, but includes trailing. Never sells at a loss.
  2 = Trailing Stop-Loss. As strategy #1, but potentially sells at a loss.
  3 = Trailing Stop-Loss Short/Mid Term. As strategy #2, but does not trail
      forever. Sells as soon as ticker >= mult.
  4 = Stop-Loss. No trailing. As strategy #0, but potentially sells at a loss.

Notify:
  0 = nothing, ever
  1 = errors only
  2 = errors + filled orders (default)
  3 = everything (including opened and cancelled orders)
`
	return strings.TrimSpace(text)
}

func (c *SellCommand) Synopsis() string {
	return "Automatically opens new sell orders on the specified exchange/market."
}
