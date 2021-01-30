### Nefertiti ###

Nefertiti is a FREE crypto trading bot that follows a simple but proven trading strategy; buy the dip and then sell those trades as soon as possible.

Original project: https://github.com/svanas/nefertiti

### Exchanges ###

The trading bot supports, for now, only Binance exchange

### Running ###

#### Command list
```bash
go build
./nefertiti --help
```

#### Running looking for low prices for BTC or USDT pairs

```bash
go build
API_KEY=$BINANCE_API_KEY API_SECRET=$BINANCE_API_SECRET PUSHOVER_APP_KEY=$PUSHOVER_APP_KEY PUSHOVER_USER_KEY=$PUSHOVER_USER_KEY ./nefertiti.sh markets &
API_KEY=$BINANCE_API_KEY API_SECRET=$BINANCE_API_SECRET PUSHOVER_APP_KEY=$PUSHOVER_APP_KEY PUSHOVER_USER_KEY=$PUSHOVER_USER_KEY ./nefertiti.sh sell
```
