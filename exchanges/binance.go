package exchanges

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	filemutex "github.com/alexflint/go-filemutex"
	"github.com/go-errors/errors"
	"github.com/marioarranzr/nefertiti/flag"
	"github.com/marioarranzr/nefertiti/model"
	"github.com/marioarranzr/nefertiti/notify"
	"github.com/marioarranzr/nefertiti/pricing"
	"github.com/marioarranzr/nefertiti/session"
	"github.com/marioarranzr/nefertiti/uuid"
	exchange "github.com/svanas/go-binance"
)

var (
	binanceMutex *filemutex.FileMutex
)

const (
	binanceSessionFile = "binance.time"
	binanceSessionLock = "binance.lock"
)

//-------------------- globals -------------------

func init() {
	exchange.BeforeRequest = func(client *exchange.Client, endpoint string, weight int) error {
		var err error

		if binanceMutex == nil {
			if binanceMutex, err = filemutex.New(session.GetSessionFile(binanceSessionLock)); err != nil {
				return err
			}
		}

		if err = binanceMutex.Lock(); err != nil {
			return err
		}

		var lastRequest *time.Time
		if lastRequest, err = session.GetLastRequest(binanceSessionFile); err != nil {
			return err
		}

		if lastRequest != nil {
			var rps float64
			if rps, err = client.GetRequestsPerSecond(weight); err != nil {
				return err
			}
			elapsed := time.Since(*lastRequest)
			if elapsed.Seconds() < (float64(1) / rps) {
				sleep := time.Duration((float64(time.Second) / rps)) - elapsed
				if flag.Debug() {
					log.Printf("[DEBUG] sleeping %f seconds", sleep.Seconds())
				}
				time.Sleep(sleep)
			}
		}

		if flag.Debug() {
			log.Println("[DEBUG] GET " + endpoint)
		}

		return nil
	}
	exchange.AfterRequest = func() {
		defer func() {
			binanceMutex.Unlock()
		}()
		session.SetLastRequest(binanceSessionFile, time.Now())
	}
}

func binanceOrderSide(order *exchange.Order) model.OrderSide {
	if order.Side == string(exchange.SideTypeBuy) {
		return model.BUY
	} else {
		if order.Side == string(exchange.SideTypeSell) {
			return model.SELL
		}
	}
	return model.ORDER_SIDE_NONE
}

func binanceOrderType(order *exchange.Order) model.OrderType {
	if order.Type == string(exchange.OrderTypeLimit) || order.Type == string(exchange.OrderTypeStopLossLimit) {
		return model.LIMIT
	} else {
		if order.Type == string(exchange.OrderTypeMarket) || order.Type == string(exchange.OrderTypeStopLoss) {
			return model.MARKET
		}
	}
	return model.ORDER_TYPE_NONE
}

func binanceOrderIndex(orders []*exchange.Order, orderId int64) int {
	for i, o := range orders {
		if o.OrderId == orderId {
			return i
		}
	}
	return -1
}

func binanceOrderIsOCO(orders []*exchange.Order, order1 *exchange.Order) bool {
	if order1.Type == string(exchange.OrderTypeStopLoss) || order1.Type == string(exchange.OrderTypeStopLossLimit) || order1.Type == string(exchange.OrderTypeLimitMaker) {
		for _, order2 := range orders {
			if order2.OrderId != order1.OrderId {
				if order2.Side == order1.Side && order2.Symbol == order1.Symbol && order2.OrigQuantity == order1.OrigQuantity {
					if order2.Type == string(exchange.OrderTypeStopLoss) || order2.Type == string(exchange.OrderTypeStopLossLimit) {
						return order1.Type == string(exchange.OrderTypeLimitMaker)
					}
					if order2.Type == string(exchange.OrderTypeLimitMaker) {
						return order1.Type == string(exchange.OrderTypeStopLoss) || order1.Type == string(exchange.OrderTypeStopLossLimit)
					}
				}
			}
		}
	}
	return false
}

//---------------- BinanceOrderEx ----------------

type BinanceOrderEx struct {
	*exchange.Order
}

func (order *BinanceOrderEx) MarshalJSON() ([]byte, error) {
	type (
		Alias BinanceOrderEx
	)

	omd := model.ParseMetaData(order.ClientOrderId)
	if omd.Price > 0 && omd.Mult > 0 {
		return json.Marshal(&struct {
			*Alias
			Meta string `json:"metadata"`
		}{
			Alias: (*Alias)(order),
			Meta:  fmt.Sprintf("bought at: %.8f, mult: %.2f", omd.Price, omd.Mult),
		})
	}

	return json.Marshal(&struct{ *Alias }{Alias: (*Alias)(order)})
}

func getBinanceOrderEx(order *exchange.Order) (*BinanceOrderEx, error) {
	var (
		err error
		buf []byte
		out BinanceOrderEx
	)
	if buf, err = json.Marshal(order); err != nil {
		return nil, errors.Wrap(err, 1)
	}
	if err = json.Unmarshal(buf, &out); err != nil {
		return nil, errors.Wrap(err, 1)
	}
	return &out, nil
}

func binanceOrderToString(order *exchange.Order) ([]byte, error) {
	var (
		err error
		out []byte
		new *BinanceOrderEx
	)
	if new, err = getBinanceOrderEx(order); err != nil {
		return nil, err
	}
	if out, err = json.Marshal(new); err != nil {
		return nil, errors.Wrap(err, 1)
	}
	return out, nil
}

//-------------------- Binance -------------------

type Binance struct {
	*model.ExchangeInfo
}

//-------------------- private -------------------

func (self *Binance) futures() bool {
	return flag.Exists("futures")
}

// The broker ID should be sent as the initial part in the "newClientOrderId" when your
// client places an order so that our will consider the order as a brokerage order. For
// example, if your broker ID is  “ABC123”, the newClientOrderId should be started with
// "x-ABC123" when any of your clients places an order.
func (self *Binance) getBrokerId() string {
	const (
		MY_BROKER_ID = "J6MCRYME"
	)
	out := uuid.New().LongEx("") // the clientOrderId cannot have more than 32 characters.
	out = fmt.Sprintf("x-%s", MY_BROKER_ID) + out[10:]
	return out
}

func (self *Binance) report(err error) {
	log.Printf("[ERROR] %v", err)
}

func (self *Binance) notify(err error, level int64, service model.Notify) {
	msg := fmt.Sprintf("%v", err)
	_, ok := err.(*errors.Error)
	if ok {
		log.Printf("[ERROR] %s", err.(*errors.Error).ErrorStack())
	} else {
		log.Printf("[ERROR] %s", msg)
	}

	if service != nil {
		if notify.CanSend(level, notify.ERROR) {
			err := service.SendMessage(msg, "Binance - ERROR")
			if err != nil {
				self.report(err)
			}
		}
	}
}

func (self *Binance) getMinTrade(client *exchange.Client, market string, cached bool) (float64, error) {
	precs, err := exchange.GetPrecs(client, cached)
	if err != nil {
		return 0, errors.Wrap(err, 1)
	}
	prec := precs.PrecFromSymbol(market)
	if prec != nil {
		return prec.Min, nil
	}
	return 0, nil
}

//-------------------- public --------------------

func (self *Binance) GetInfo() *model.ExchangeInfo {
	return self.ExchangeInfo
}

func (self *Binance) GetClient(private, sandbox bool) (interface{}, error) {
	if !private {
		return exchange.NewClient("", "", self.futures()), nil
	}

	apiKey, apiSecret, err := promptForApiKeys(Binance_)
	if err != nil {
		return nil, err
	}

	return exchange.NewClient(apiKey, apiSecret, self.futures()), nil
}

func (self *Binance) GetMarkets(cached, sandbox bool) ([]model.Market, error) {
	var out []model.Market

	precs, err := exchange.GetPrecs(exchange.NewClient("", "", self.futures()), cached)

	if err != nil {
		return nil, errors.Wrap(err, 1)
	}

	for _, prec := range precs {
		if prec.Status == string(exchange.SymbolStatusTrading) {
			out = append(out, model.Market{
				Name:  prec.Symbol,
				Base:  prec.Base,
				Quote: prec.Quote,
			})
		}
	}

	return out, nil
}

func (self *Binance) GetMarketsEx(cached, sandbox bool, quotes []string) ([]model.Market, error) {
	markets, err := self.GetMarkets(cached, sandbox)

	if err != nil {
		return nil, err
	}

	if len(quotes) == 0 {
		return markets, err
	}

	var out []model.Market
	for _, market := range markets {
		for _, quote := range quotes {
			if strings.EqualFold(market.Quote, quote) {
				out = append(out, market)
			}
		}
	}

	return out, nil
}

func (self *Binance) FormatMarket(base, quote string) string {
	return strings.ToUpper(base + quote)
}

// listens to the open orders, send a notification on newly opened orders.
func (self *Binance) listen(client *exchange.Client, service model.Notify, level int64, old []*exchange.Order) ([]*exchange.Order, error) {
	var err error

	// get my open orders
	var new []*exchange.Order
	if new, err = client.NewListOpenOrdersService().Do(context.Background()); err != nil {
		return old, errors.Wrap(err, 1)
	}

	// look for new orders
	for _, order := range new {
		if binanceOrderIndex(old, order.OrderId) == -1 {
			var data []byte
			if data, err = binanceOrderToString(order); err != nil {
				return new, err
			}

			log.Println("[OPEN] " + string(data))

			if service != nil {
				side := binanceOrderSide(order)
				if side != model.ORDER_SIDE_NONE {
					if notify.CanSend(level, notify.OPENED) || (level == notify.LEVEL_DEFAULT && side == model.SELL) {
						if err = service.SendMessage(string(data), ("Binance - Open " + model.FormatOrderSide(side))); err != nil {
							self.report(err)
						}
					}
				}
			}
		}
	}

	return new, nil
}

// listens to the filled orders, look for newly filled orders, automatically place new sell orders.
func (self *Binance) sell(
	client *exchange.Client,
	strategy model.Strategy,
	quotes []string,
	mult1 float64,
	hold model.Markets,
	service model.Notify,
	twitter *notify.TwitterKeys,
	level int64,
	old []*exchange.Order,
	sandbox bool,
	debug bool,
) ([]*exchange.Order, error) {
	var err error

	// get my filled orders
	var (
		new     []*exchange.Order
		markets []model.Market
	)
	if markets, err = self.GetMarketsEx(true, sandbox, quotes); err != nil {
		return old, err
	}
	for _, market := range markets {
		var orders []*exchange.Order
		if orders, err = client.NewListOrdersService().Symbol(market.Name).Do(context.Background()); err != nil {
			return old, errors.Wrap(err, 1)
		}
		for _, order := range orders {
			if order.Status == "FILLED" {
				new = append(new, order)
			}
		}
	}

	// look for newly filled orders
	for _, order := range new {
		if binanceOrderIndex(old, order.OrderId) == -1 {
			side := binanceOrderSide(order)

			var data []byte
			if data, err = binanceOrderToString(order); err != nil {
				return new, err
			}
			log.Println("[FILLED] " + string(data))

			// has a stop loss been filled? then place a buy order double the order size
			if side == model.SELL {
				if strategy == model.STRATEGY_STOP_LOSS {
					if order.Type == string(exchange.OrderTypeStopLoss) || order.Type == string(exchange.OrderTypeStopLossLimit) {
						if model.ParseMetaData(order.ClientOrderId).Trail {
							// do not mistakenly re-buy trailing profit orders that were filled
						} else {
							var ticker float64
							if ticker, err = self.GetTicker(client, order.Symbol); err == nil {
								var prec int
								if prec, err = self.GetPricePrec(client, order.Symbol); err == nil {
									adjusted := order.GetStopPrice() * 1.01
									if ticker > pricing.RoundToPrecision(adjusted, prec) {
										log.Printf("[INFO] Not re-buying %s because ticker %.8f is higher than stop price %s\n", order.Symbol, ticker, order.StopPrice)
									} else {
										size := 2 * order.GetSize()
										if prec, err = self.GetSizePrec(client, order.Symbol); err == nil {
											_, _, err = self.Order(client,
												model.BUY,
												order.Symbol,
												pricing.RoundToPrecision(size, prec),
												0, model.MARKET, "",
											)
										}
									}
								}
							}
							if err != nil {
								return new, errors.Errorf("%v %v %v", err, "\t", string(data))
							}
						}
					}
				}
			}

			if side != model.ORDER_SIDE_NONE {
				// send notification(s)
				if notify.CanSend(level, notify.FILLED) {
					if service != nil {
						title := fmt.Sprintf("Binance - Done %s", model.FormatOrderSide(side))
						if side == model.SELL {
							if model.HasMetaData(order.ClientOrderId) {
								metadata := model.ParseMetaData(order.ClientOrderId)
								if binanceOrderType(order) == model.MARKET {
									title = fmt.Sprintf("%s %.2f%%", title, ((metadata.Mult - 1) * 100))
								} else {
									old := metadata.Price
									new := order.GetPrice()
									if new > 0 {
										title = fmt.Sprintf("%s %.2f%%", title, (((new - old) / old) * 100))
									}
								}
							}
						}
						if err = service.SendMessage(string(data), title); err != nil {
							self.report(err)
						}
					}
					if twitter != nil {
						notify.Tweet(twitter, fmt.Sprintf("Done %s. %s priced at %s #Binance", model.FormatOrderSide(side), model.TweetMarket(markets, order.Symbol), order.Price))
					}
				}
				// has a buy order been filled? then place a sell order
				if side == model.BUY {
					var (
						temp string
						call *model.Call
					)
					temp = session.GetTempFileName(order.ClientOrderId, ".binance")
					if call, err = model.File2Call(temp); err == nil {
						defer func() {
							os.Remove(temp)
						}()
					}
					// --- BEGIN --- svanas 2018-05-10 --- <APIError> code=-1013, msg=Invalid price.
					bought := order.GetPrice()
					if bought == 0 {
						if call != nil {
							bought = call.Price
						}
						if bought == 0 {
							if bought, err = self.GetTicker(client, order.Symbol); err != nil {
								return new, err
							}
						}
					}
					// ---- END ---- svanas 2018-05-10 ---------------------------------------------
					var (
						base  string
						quote string
					)
					base, quote, err = model.ParseMarket(markets, order.Symbol)
					if err == nil {
						qty := self.GetMaxSize(client, base, quote, hold.HasMarket(order.Symbol), order.GetSize())
						if qty > 0 {
							mult2 := mult1
							if call != nil && call.HasTarget() {
								if strategy == model.STRATEGY_STANDARD || strategy == model.STRATEGY_TRAILING_STOP_LOSS_QUICK || strategy == model.STRATEGY_STOP_LOSS {
									mult2 = pricing.FloorToPrecision((call.ParseTarget() / bought), 2)
								}
							}
							var prec int
							if prec, err = self.GetPricePrec(client, order.Symbol); err == nil {
								if strategy == model.STRATEGY_TRAILING_STOP_LOSS || strategy == model.STRATEGY_TRAILING_STOP_LOSS_QUICK || strategy == model.STRATEGY_STOP_LOSS {
									var ticker float64
									if ticker, err = self.GetTicker(client, order.Symbol); err == nil {
										handled := false
										target1 := pricing.Multiply(bought, mult1, prec)
										if call != nil && call.HasTarget() {
											target1 = pricing.RoundToPrecision(call.ParseTarget(), prec)
										}
										if strategy == model.STRATEGY_TRAILING_STOP_LOSS_QUICK || strategy == model.STRATEGY_STOP_LOSS {
											if ticker >= target1 {
												_, _, err = self.Order(client,
													model.SELL,
													order.Symbol,
													order.GetSize(),
													0, model.MARKET,
													model.NewMetaData(bought, mult2).String(),
												)
												handled = true
											}
										}
										if !handled {
											var stop float64
											if strategy == model.STRATEGY_STOP_LOSS {
												stop = ticker / pricing.NewMult(mult1, 2.0)
											} else {
												stop = ticker / pricing.NewMult(mult1, 0.5)
											}
											if call != nil && call.HasStop() {
												stop = call.ParseStop()
											}
											stop = pricing.RoundToPrecision(stop, prec)
											// non-trailing stop loss? place an OCO (aka One-Cancels-the-Other)
											if strategy == model.STRATEGY_STOP_LOSS {
												var precs exchange.Precs
												precs, err = exchange.GetPrecs(client, true)
												if err == nil {
													prec := precs.PrecFromSymbol(order.Symbol)
													if prec != nil && prec.OCO {
														if _, err = self.OCO(client,
															model.SELL,
															order.Symbol,
															qty,
															target1,
															stop,
															model.NewMetaDataEx(bought, mult2, false).String(),
															model.NewMetaDataEx(bought, mult2, false).String(),
														); err != nil {
															self.report(err)
														}
														handled = err == nil
													}
												}
											}
											if !handled {
												_, err = self.StopLoss(client,
													order.Symbol,
													qty,
													stop,
													model.MARKET,
													model.NewMetaDataEx(bought, mult2, strategy != model.STRATEGY_STOP_LOSS).String(),
												)
											}
										}
									}
								} else {
									_, _, err = self.Order(client,
										model.SELL,
										order.Symbol,
										qty,
										pricing.Multiply(bought, mult2, prec),
										model.LIMIT,
										model.NewMetaData(bought, mult2).String(),
									)
								}
							}
						}
					}
					if err != nil {
						return new, errors.Errorf("%v %v %v", err, "\t", string(data))
					}
				}
			}
		}
	}

	return new, nil
}

func (self *Binance) Sell(
	start time.Time,
	hold model.Markets,
	sandbox, tweet, debug bool,
	success model.OnSuccess,
) error {
	var err error

	var (
		apiKey    string
		apiSecret string
	)
	if apiKey, apiSecret, err = promptForApiKeys(Binance_); err != nil {
		return err
	}

	var service model.Notify = nil
	if service, err = notify.New().Init(flag.Interactive(), true); err != nil {
		return err
	}

	var twitter *notify.TwitterKeys = nil
	if tweet {
		if twitter, err = notify.TwitterPromptForKeys(flag.Interactive()); err != nil {
			return err
		}
	}

	client := exchange.NewClient(apiKey, apiSecret, self.futures())

	// get my open orders
	var open []*exchange.Order
	if open, err = client.NewListOpenOrdersService().Do(context.Background()); err != nil {
		return errors.Wrap(err, 1)
	}

	// get my filled orders
	var (
		quotes  []string = []string{model.BTC}
		filled  []*exchange.Order
		markets []model.Market
	)
	flg := flag.Get("quote")
	if flg.Exists {
		quotes = strings.Split(flg.String(), ",")
	} else {
		flag.Set("quote", strings.Join(quotes, ","))
	}
	if markets, err = self.GetMarketsEx(true, sandbox, quotes); err != nil {
		return err
	}
	for _, market := range markets {
		var orders []*exchange.Order
		if orders, err = client.NewListOrdersService().Symbol(market.Name).Do(context.Background()); err != nil {
			return errors.Wrap(err, 1)
		}
		for _, order := range orders {
			if order.Status == "FILLED" {
				filled = append(filled, order)
			}
		}
	}

	if err = success(service); err != nil {
		return err
	}

	for {
		// read the dynamic settings
		var (
			level    int64          = notify.Level()
			strategy model.Strategy = model.GetStrategy()
			quotes   []string       = strings.Split(flag.Get("quote").String(), ",")
		)
		// listen to the filled orders, look for newly filled orders, automatically place new sell orders.
		filled, err = self.sell(client, strategy, quotes, model.GetMult(), hold, service, twitter, level, filled, sandbox, debug)
		if err != nil {
			self.notify(err, level, service)
		} else {
			// listen to the open orders, send a notification on newly opened orders.
			open, err = self.listen(client, service, level, open)
			if err != nil {
				self.notify(err, level, service)
			} else {
				// listen to the open orders, follow up on the trailing stop loss strategy
				if strategy == model.STRATEGY_TRAILING || strategy == model.STRATEGY_TRAILING_STOP_LOSS || strategy == model.STRATEGY_TRAILING_STOP_LOSS_QUICK || strategy == model.STRATEGY_STOP_LOSS {
					cache := make(map[string]float64)
					for _, order := range open {
						// enumerate over stop loss orders
						if order.Type == string(exchange.OrderTypeStopLoss) || order.Type == string(exchange.OrderTypeStopLossLimit) {
							var prec int
							if prec, err = self.GetPricePrec(client, order.Symbol); err == nil {
								handled := false
								if strategy == model.STRATEGY_TRAILING_STOP_LOSS_QUICK || strategy == model.STRATEGY_STOP_LOSS {
									if !binanceOrderIsOCO(open, order) {
										ticker, ok := cache[order.Symbol]
										if !ok {
											if ticker, err = self.GetTicker(client, order.Symbol); err == nil {
												cache[order.Symbol] = ticker
											}
										}
										if ticker > 0 {
											if model.HasMetaData(order.ClientOrderId) {
												metadata := model.ParseMetaData(order.ClientOrderId)
												if ticker >= pricing.Multiply(metadata.Price, metadata.Mult, prec) {
													var data []byte
													if data, err = json.Marshal(order); err == nil {
														log.Println("[CANCELLED] " + string(data))
													}
													if _, err = client.NewCancelOrderService().Symbol(order.Symbol).OrderId(order.OrderId).Do(context.Background()); err == nil {
														_, _, err = self.Order(client,
															model.SELL,
															order.Symbol,
															order.GetSize(),
															0, model.MARKET,
															model.NewMetaData(metadata.Price, metadata.Mult).String(),
														)
														handled = true
													}
												}
											}
										}
									}
								}
								if !handled {
									if model.HasMetaData(order.ClientOrderId) {
										handled = !model.ParseMetaData(order.ClientOrderId).Trail
									} else {
										handled = strategy == model.STRATEGY_STOP_LOSS
									}
									if !handled {
										ticker, ok := cache[order.Symbol]
										if !ok {
											if ticker, err = self.GetTicker(client, order.Symbol); err == nil {
												cache[order.Symbol] = ticker
											}
										}
										if ticker > 0 {
											var (
												mult float64
												stop float64
											)
											mult = model.GetOrderMult(order.ClientOrderId)
											if strategy == model.STRATEGY_STOP_LOSS {
												stop = ticker / pricing.NewMult(mult, 2.0)
											} else {
												stop = ticker / pricing.NewMult(mult, 0.5)
											}
											// is the distance bigger than mult? then cancel the stop loss, and place a new one.
											if order.GetStopPrice() < pricing.RoundToPrecision(stop, prec) {
												var data []byte
												if data, err = json.Marshal(order); err == nil {
													log.Println("[CANCELLED] " + string(data))
												}
												if _, err = client.NewCancelOrderService().Symbol(order.Symbol).OrderId(order.OrderId).Do(context.Background()); err == nil {
													_, err = self.StopLoss(client,
														order.Symbol,
														order.GetSize(),
														pricing.RoundToPrecision(stop, prec),
														binanceOrderType(order),
														model.ParseMetaData(order.ClientOrderId).String(),
													)
												}
											}
										}
									}
								}
							}
							if err != nil {
								var data []byte
								if data, _ = binanceOrderToString(order); data == nil {
									self.notify(err, level, service)
								} else {
									self.notify(errors.Errorf("%v %v %v", err, "\t", string(data)), level, service)
								}
							}

						}
						// enumerate over limit sell orders
						if order.Type == string(exchange.OrderTypeLimit) {
							side := binanceOrderSide(order)
							if side == model.SELL {
								if strategy != model.STRATEGY_STOP_LOSS {
									ticker, ok := cache[order.Symbol]
									if !ok {
										if ticker, err = self.GetTicker(client, order.Symbol); err == nil {
											cache[order.Symbol] = ticker
										}
									}
									if ticker > 0 {
										var (
											mult  float64 = model.GetOrderMult(order.ClientOrderId)
											price float64 = pricing.NewMult(mult, 0.75) * (order.GetPrice() / mult)
										)
										// is the ticker nearing the price? then cancel the limit sell order, and place a stop loss order below the ticker.
										if ticker > price {
											var prec int
											if prec, err = self.GetPricePrec(client, order.Symbol); err == nil {
												price = pricing.NewMult(mult, 0.5) * (ticker / mult)
												if ticker > pricing.RoundToPrecision(price, prec) { // <APIError> code=-2010, msg=Order would trigger immediately.
													var data []byte
													if data, err = json.Marshal(order); err == nil {
														log.Println("[CANCELLED] " + string(data))
													}
													if _, err = client.NewCancelOrderService().Symbol(order.Symbol).OrderId(order.OrderId).Do(context.Background()); err == nil {
														_, err = self.StopLoss(client,
															order.Symbol,
															order.GetSize(),
															pricing.RoundToPrecision(price, prec),
															model.MARKET,
															model.ParseMetaData(order.ClientOrderId).String(),
														)
														// --- BEGIN --- svanas 2018-12-01 --- reopen the above limit sell on an API error ---
														if err != nil {
															_, ok := err.(*exchange.BinanceError)
															if ok {
																self.report(err)
																_, _, err = self.Order(client,
																	binanceOrderSide(order),
																	order.Symbol,
																	order.GetSize(),
																	order.GetPrice(),
																	model.LIMIT,
																	model.ParseMetaData(order.ClientOrderId).String(),
																)
															}
														}
														// ---- END ---- svanas 2018-12-01 ---------------------------------------------------
													}
												}
											}
											if err != nil {
												var data []byte
												if data, _ = binanceOrderToString(order); data == nil {
													self.notify(err, level, service)
												} else {
													self.notify(errors.Errorf("%v %v %v", err, "\t", string(data)), level, service)
												}
											}
										}
									}
								}
							}
						}
					}
				}
			}
		}
	}

	return nil
}

func (self *Binance) Order(
	client interface{},
	side model.OrderSide,
	market string,
	size float64,
	price float64,
	kind model.OrderType,
	meta string,
) (oid []byte, raw []byte, err error) {
	binance, ok := client.(*exchange.Client)
	if !ok {
		return nil, nil, errors.New("invalid argument: client")
	}

	var service *exchange.CreateOrderService
	service = binance.NewCreateOrderService().Symbol(market).Quantity(strconv.FormatFloat(size, 'f', -1, 64))

	if meta != "" {
		service.NewClientOrderId(meta)
	} else {
		service.NewClientOrderId(self.getBrokerId())
	}

	if kind == model.MARKET {
		service.Type(exchange.OrderTypeMarket)
	} else {
		service.Type(exchange.OrderTypeLimit).TimeInForce(exchange.TimeInForceGTC).Price(strconv.FormatFloat(price, 'f', -1, 64))
	}

	if side == model.BUY {
		service.Side(exchange.SideTypeBuy)
	} else if side == model.SELL {
		service.Side(exchange.SideTypeSell)
	}

	var order *exchange.CreateOrderResponse
	if order, err = service.Do(context.Background()); err != nil {
		return nil, nil, errors.Wrap(err, 1)
	}

	var out []byte
	if out, err = json.Marshal(order); err != nil {
		return nil, nil, errors.Wrap(err, 1)
	}

	return []byte(order.ClientOrderId), out, nil
}

func (self *Binance) StopLoss(client interface{}, market string, size float64, price float64, kind model.OrderType, meta string) ([]byte, error) {
	var err error

	binance, ok := client.(*exchange.Client)
	if !ok {
		return nil, errors.New("invalid argument: client")
	}

	var service *exchange.CreateOrderService
	service = binance.NewCreateOrderService()

	service.Symbol(market).Side(exchange.SideTypeSell).Quantity(strconv.FormatFloat(size, 'f', -1, 64)).StopPrice(strconv.FormatFloat(price, 'f', -1, 64))

	if kind == model.MARKET {
		service.Type(exchange.OrderTypeStopLoss)
	} else {
		var prec int
		if prec, err = self.GetPricePrec(client, market); err != nil {
			return nil, err
		}
		limit := price
		for true {
			limit = limit * 0.99
			if pricing.RoundToPrecision(limit, prec) < price {
				break
			}
		}
		service.Type(exchange.OrderTypeStopLossLimit).TimeInForce(exchange.TimeInForceGTC).Price(strconv.FormatFloat(pricing.RoundToPrecision(limit, prec), 'f', -1, 64))
	}

	if meta != "" {
		service.NewClientOrderId(meta)
	} else {
		service.NewClientOrderId(self.getBrokerId())
	}

	var order *exchange.CreateOrderResponse
	if order, err = service.Do(context.Background()); err != nil {
		// --- BEGIN --- svanas 2019-02-07 ------------------------------------
		_, ok := err.(*exchange.BinanceError)
		if ok {
			self.report(err)
			// -1013 stop loss orders are not supported for this symbol
			if kind != model.LIMIT {
				return self.StopLoss(client, market, size, price, model.LIMIT, meta)
			}
			// -2010 order would trigger immediately
			if strings.Contains(err.Error(), "would trigger immediately") {
				var prec int
				if prec, err = self.GetPricePrec(client, market); err == nil {
					lower := price
					for true {
						lower = lower * 0.99
						if pricing.RoundToPrecision(lower, prec) < price {
							break
						}
					}
					return self.StopLoss(client, market, size, pricing.RoundToPrecision(lower, prec), kind, meta)
				}
			}
			// -2010 Filter failure: MAX_NUM_ALGO_ORDERS
			if strings.Contains(err.Error(), "MAX_NUM_ALGO_ORDERS") {
				if model.HasMetaData(meta) {
					omd := model.ParseMetaData(meta)
					var prec int
					if prec, err = self.GetPricePrec(client, market); err == nil {
						var out []byte
						_, out, err = self.Order(client, model.SELL, market, size, pricing.Multiply(omd.Price, omd.Mult, prec), model.LIMIT, meta)
						return out, nil
					}
				}
			}
		}
		// ---- END ---- svanas 2019-02-07 ------------------------------------
		return nil, errors.Wrap(err, 1)
	}

	var out []byte
	if out, err = json.Marshal(order); err != nil {
		return nil, errors.Wrap(err, 1)
	}

	return out, nil
}

func (self *Binance) OCO(client interface{}, side model.OrderSide, market string, size float64, price, stop float64, meta1, meta2 string) ([]byte, error) {
	var (
		err  error
		svc  *exchange.CreateOcoService
		resp *exchange.CreateOcoOrdersResponse
	)

	binance, ok := client.(*exchange.Client)
	if !ok {
		return nil, errors.New("invalid argument: client")
	}

	svc = binance.NewCreateOcoService().Symbol(market).Quantity(size).Price(price).StopPrice(stop)

	if side == model.BUY {
		svc.Side(exchange.SideTypeBuy)
	} else if side == model.SELL {
		svc.Side(exchange.SideTypeSell)
	}

	if meta1 != "" {
		svc.StopClientOrderId(meta1)
	} else {
		svc.StopClientOrderId(self.getBrokerId())
	}
	if meta2 != "" {
		svc.LimitClientOrderId(meta2)
	} else {
		svc.LimitClientOrderId(self.getBrokerId())
	}

	if resp, err = svc.Do(context.Background()); err != nil {
		_, ok := err.(*exchange.BinanceError)
		if ok {
			self.report(err)
			// -2010 Filter failure: MAX_NUM_ALGO_ORDERS
			if strings.Contains(err.Error(), "MAX_NUM_ALGO_ORDERS") {
				var out []byte
				_, out, err = self.Order(client, side, market, size, price, model.LIMIT, meta2)
				if err != nil {
					return nil, err
				} else {
					return out, nil
				}
			}
			// -1013 Stop loss orders are not supported for this symbol
			if strings.Contains(err.Error(), "loss orders are not supported") {
				var prec int
				if prec, err = self.GetPricePrec(client, market); err != nil {
					return nil, err
				}
				lower := stop
				for true {
					lower = lower * 0.99
					if pricing.RoundToPrecision(lower, prec) < stop {
						break
					}
				}
				svc.StopLimitPrice(pricing.RoundToPrecision(lower, prec))
				resp, err = svc.Do(context.Background())
			}
		}
		if err != nil {
			return nil, errors.Wrap(err, 1)
		}
	}

	var out []byte
	if out, err = json.Marshal(resp); err != nil {
		return nil, errors.Wrap(err, 1)
	}

	return out, nil
}

func (self *Binance) GetClosed(client interface{}, market string) (model.Orders, error) {
	var err error

	binance, ok := client.(*exchange.Client)
	if !ok {
		return nil, errors.New("invalid argument: client")
	}

	var orders []*exchange.Order
	if orders, err = binance.NewListOrdersService().Symbol(market).Do(context.Background()); err != nil {
		return nil, errors.Wrap(err, 1)
	}

	var out model.Orders
	for _, order := range orders {
		if order.Status == "FILLED" {
			out = append(out, model.Order{
				Side:      binanceOrderSide(order),
				Market:    order.Symbol,
				Size:      order.GetSize(),
				Price:     order.GetPrice(),
				CreatedAt: time.Unix(order.Time/1000, 0),
			})
		}
	}

	return out, nil
}

func (self *Binance) GetOpened(client interface{}, market string) (model.Orders, error) {
	var err error

	binance, ok := client.(*exchange.Client)
	if !ok {
		return nil, errors.New("invalid argument: client")
	}

	var orders []*exchange.Order
	if orders, err = binance.NewListOpenOrdersService().Symbol(market).Do(context.Background()); err != nil {
		return nil, errors.Wrap(err, 1)
	}

	var out model.Orders
	for _, order := range orders {
		out = append(out, model.Order{
			Side:      binanceOrderSide(order),
			Market:    order.Symbol,
			Size:      order.GetSize(),
			Price:     order.GetPrice(),
			CreatedAt: time.Unix(order.Time/1000, 0),
		})
	}

	return out, nil
}

func (self *Binance) GetBook(client interface{}, market string, side model.BookSide) (interface{}, error) {
	var err error

	binance, ok := client.(*exchange.Client)
	if !ok {
		return nil, errors.New("invalid argument: client")
	}

	var book *exchange.DepthResponse
	if book, err = binance.NewDepthService().Symbol(market).Limit(1000).Do(context.Background()); err != nil {
		return nil, errors.Wrap(err, 1)
	}

	var out []exchange.BookEntry
	if side == model.BOOK_SIDE_ASKS {
		out = book.Asks
	} else {
		out = book.Bids
	}

	return out, nil
}

func (self *Binance) Aggregate(client, book interface{}, market string, agg float64) (model.Book, error) {
	bids, ok := book.([]exchange.BookEntry)
	if !ok {
		return nil, errors.New("invalid argument: book")
	}

	prec, err := self.GetPricePrec(client, market)
	if err != nil {
		return nil, err
	}

	var out model.Book
	for _, e := range bids {
		price := pricing.RoundToPrecision(pricing.RoundToNearest(e.Price(), agg), prec)
		entry := out.EntryByPrice(price)
		if entry != nil {
			entry.Size = entry.Size + e.Quantity()
		} else {
			entry = &model.BookEntry{
				Buy: &model.Buy{
					Market: market,
					Price:  price,
				},
				Size: e.Quantity(),
			}
			out = append(out, *entry)
		}
	}

	return out, nil
}

func (self *Binance) GetTicker(client interface{}, market string) (float64, error) {
	var err error

	binance, ok := client.(*exchange.Client)
	if !ok {
		return 0, errors.New("invalid argument: client")
	}

	var ticker *exchange.PriceChangeStats
	if ticker, err = binance.NewPriceChangeStatsService().Symbol(market).Do(context.Background()); err != nil {
		return 0, errors.Wrap(err, 1)
	}

	var out float64
	if out, err = strconv.ParseFloat(ticker.LastPrice, 64); err != nil {
		return 0, errors.Wrap(err, 1)
	}

	return out, nil
}

func (self *Binance) Get24h(client interface{}, market string) (*model.Stats, error) {
	var err error

	binance, ok := client.(*exchange.Client)
	if !ok {
		return nil, errors.New("invalid argument: client")
	}

	var stats *exchange.PriceChangeStats
	if stats, err = binance.NewPriceChangeStatsService().Symbol(market).Do(context.Background()); err != nil {
		return nil, errors.Wrap(err, 1)
	}

	var high float64
	if high, err = strconv.ParseFloat(stats.HighPrice, 64); err != nil {
		return nil, errors.Wrap(err, 1)
	}

	var low float64
	if low, err = strconv.ParseFloat(stats.LowPrice, 64); err != nil {
		return nil, errors.Wrap(err, 1)
	}

	var volume float64
	if volume, err = strconv.ParseFloat(stats.QuoteVolume, 64); err != nil {
		return nil, errors.Wrap(err, 1)
	}

	return &model.Stats{
		Market:    market,
		High:      high,
		Low:       low,
		BtcVolume: volume,
	}, nil
}

func (self *Binance) GetPricePrec(client interface{}, market string) (int, error) {
	binance, ok := client.(*exchange.Client)
	if !ok {
		return 0, errors.New("invalid argument: client")
	}
	precs, err := exchange.GetPrecs(binance, true)
	if err != nil {
		return 0, errors.Wrap(err, 1)
	}
	prec := precs.PrecFromSymbol(market)
	if prec != nil {
		return prec.Price, nil
	}
	return 8, nil
}

func (self *Binance) GetSizePrec(client interface{}, market string) (int, error) {
	binance, ok := client.(*exchange.Client)
	if !ok {
		return 0, errors.New("invalid argument: client")
	}
	precs, err := exchange.GetPrecs(binance, true)
	if err != nil {
		return 0, errors.Wrap(err, 1)
	}
	prec := precs.PrecFromSymbol(market)
	if prec != nil {
		return prec.Size, nil
	}
	return 0, nil
}

func (self *Binance) GetMaxSize(client interface{}, base, quote string, hold bool, def float64) float64 {
	if hold {
		if base == "BNB" {
			return 0
		}
	}
	fn := func() int {
		prec, err := self.GetSizePrec(client, self.FormatMarket(base, quote))
		if err != nil {
			return 0
		} else {
			return prec
		}
	}
	return model.GetSizeMax(hold, def, fn)
}

func (self *Binance) Cancel(client interface{}, market string, side model.OrderSide) error {
	var err error

	binance, ok := client.(*exchange.Client)
	if !ok {
		return errors.New("invalid argument: client")
	}

	var orders []*exchange.Order
	if orders, err = binance.NewListOpenOrdersService().Symbol(market).Do(context.Background()); err != nil {
		return errors.Wrap(err, 1)
	}

	for _, order := range orders {
		if binanceOrderSide(order) == side {
			if _, err = binance.NewCancelOrderService().Symbol(market).OrderId(order.OrderId).Do(context.Background()); err != nil {
				return errors.Wrap(err, 1)
			}
			tmp := session.GetTempFileName(order.ClientOrderId, ".binance")
			if _, err = os.Stat(tmp); err == nil {
				os.Remove(tmp)
			}
		}
	}

	return nil
}

func (self *Binance) Buy(client interface{}, cancel bool, market string, calls model.Calls, size, deviation float64, kind model.OrderType) error {
	var err error

	binance, ok := client.(*exchange.Client)
	if !ok {
		return errors.New("invalid argument: client")
	}

	// step #1: delete the buy order(s) that are open in your book
	if cancel {
		var orders []*exchange.Order
		if orders, err = binance.NewListOpenOrdersService().Symbol(market).Do(context.Background()); err != nil {
			return errors.Wrap(err, 1)
		}
		for _, order := range orders {
			side := binanceOrderSide(order)
			if side == model.BUY {
				// do not cancel orders that we're about to re-place
				index := calls.IndexByPrice(order.GetPrice())
				if index > -1 {
					calls[index].Skip = true
				} else {
					if _, err = binance.NewCancelOrderService().Symbol(market).OrderId(order.OrderId).Do(context.Background()); err != nil {
						return errors.Wrap(err, 1)
					}
				}
			}
		}
	}

	// step 2: open the top X buy orders
	for _, call := range calls {
		if !call.Skip {
			// --- BEGIN --- svanas 2018-11-30 --- <APIError> code=-1013, msg=Filter failure: MIN_NOTIONAL.
			var (
				oid   []byte
				min   float64
				qty   float64 = size
				limit float64 = call.Price
			)
			if min, err = self.getMinTrade(binance, market, true); err != nil {
				return err
			}
			if min > 0 {
				if limit == 0 {
					if limit, err = self.GetTicker(client, market); err != nil {
						return err
					}
				}
				if (qty * limit) < min {
					var prec int
					if prec, err = self.GetSizePrec(client, market); err != nil {
						return err
					}
					qty = pricing.CeilToPrecision((min / limit), prec)
				}
			}
			// ---- END ---- svanas 2018-11-30 ------------------------------------------------------------
			if deviation > 1.0 && kind == model.LIMIT {
				var prec int
				if prec, err = self.GetPricePrec(client, market); err == nil {
					limit = pricing.RoundToPrecision((limit * deviation), prec)
				}
			}
			oid, _, err = self.Order(client,
				model.BUY,
				market,
				qty,
				limit,
				kind, "",
			)
			if err != nil {
				return err
			}
			if oid != nil {
				if kind == model.MARKET {
					var ticker float64
					if ticker, err = self.GetTicker(client, market); err == nil {
						err = model.Call2File(&model.Call{
							Buy: &model.Buy{
								Market: call.Market,
								Price:  ticker,
							},
							Stop:   call.Stop,
							Target: call.Target,
						}, session.GetTempFileName(string(oid), ".binance"))
					}
				} else {
					err = model.Call2File(&call, session.GetTempFileName(string(oid), ".binance"))
				}
				if err != nil {
					return err
				}
			}
		}
	}

	return nil
}

func NewBinance() model.Exchange {
	return &Binance{
		ExchangeInfo: &model.ExchangeInfo{
			Code: "BINA",
			Name: Binance_,
			URL:  "https://www.binance.com/",
			REST: model.Endpoint{
				URI: "https://api.binance.com",
			},
			WebSocket: model.Endpoint{
				URI: "wss://stream.binance.com:9443",
			},
			Country: "China",
		},
	}
}
