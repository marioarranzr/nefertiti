#!/bin/bash

RED="\033[0;31m"          # Red
YELLOW="\033[0;33m"       # Yellow
NC='\033[0m'              # No Color

EXCHANGE="Binance"
QUOTE="1INCH,AAVE,ADA,ADX,AERGO,AGI,AION,AKRO,ALGO,ALPHA,AMB,ANKR,ANT,APPC,ARDR,ARK,ARPA,ASR,AST,ATM,ATOM,AVA,AVAX,AXS,BAC,BAKE,BAL,BAND,BAT,BCD,BCH,BCHA,BCH,BCPT,BEAM,BEL,BKRW,BLZ,BNB,BNT,BOT,BQX,BRD,BRL,BRY,BTC,BTG,BTS,BTT,BTT,BURGER,BUSD,BVND,BZRX,CAKE,CDT,CELO,CELR,CHR,CHZ,CMT,CND,COCOS,COMP,COS,COTI,COVER,CREAM,CRV,CTK,CTSI,CTXC,CVC,CVP,DAI,DASH,DATA,DCR,DENT,DEXE,DF,DGB,DIA,DLT,DNT,DOCK,DOGE,DOT,DREP,DUSK,EASY,EGLD,ELF,ENJ,EOS,ETC,ETH,EUR,EVX,FET,FIL,FIO,FLM,FOR,FRONT,FTM,FTT,FUN,GAS,GBP,GHST,GLM,GO,GRS,GRT,GTO,GVT,GXS,HARD,HBAR,HEGIC,HIVE,HNT,HOT,ICX,IDEX,IDRT,INJ,IOST,IOTA,IOTX,IQ,IRIS,JST,JUV,KAVA,KEY,KMD,KNC,KP3R,KSM,LINK,LINK,LINKBRL,LINKNGN,LOOM,LRC,LSK,LTC,LTO,LUNA,MANA,MATIC,MBL,MDA,MDT,MFT,MITH,MKR,MTH,MTL,NANO,NAS,NAV,NBS,NCASH,NEAR,NEBL,NEO,NGN,NKN,NMR,NPXS,NULS,NXS,OAX,OCEAN,OG,OGN,OMG,ONE,ONE,ONG,ONT,ORN,OST,OXT,PERL,PHB,PIVX,PNT,POA,POLY,POWR,PPT,PROM,PSG,QKC,QLC,QSP,QTUM,RCN,RDN,REEF,REN,REP,REQ,RIF,RLC,ROSE,RSR,RT,RUB,RUNE,RVN,SAND,SC,SKL,SKY,SLP,SNGLS,SNM,SNT,SNX,SOL,SPARTA,SRM,SRM,ST,STEEM,STMX,STORJ,STPT,STRAX,STX,SUN,SUSD,SUSHI,SWRV,SXP,SXP,SXP,SYS,TCT,TFUEL,THETA,TNB,TOMO,TRB,TROY,TRU,NGN,TRX,TRY,TUSD,UAH,UMA,UNFI,UNI,USDC,USDT,UTK,VET,VIA,VIB,VIBE,VIDT,VITE,VTHO,W,WABI,WAN,WAVES,WIN,WING,WNXM,WPR,WRX,WTC,XEM,XLM,XMR,XRP,XTZ,XVG,XVS,YFI,YFII,YOYO,ZAR,ZEC,ZEN,ZIL,ZIL,ZRX"
HOLD=""
API_KEY=$API_KEY
API_SECRET=$API_SECRET
PUSHOVER_APP_KEY=$PUSHOVER_APP_KEY
PUSHOVER_USER_KEY=$PUSHOVER_USER_KEY
SIGNAL_MH_KEY="none"
SIGNAL_QS_KEY="none"
SIGNAL_CT_KEY="none"

PROFIT=1.015
PRICE=0.005

MODE="$1" # buy
ARG1="$2" # market
ARG2="$3" # coin
ARG3="$4" # coin
ARGS="--exchange=${EXCHANGE} --api-key=${API_KEY} --api-secret=${API_SECRET} --pushover-app-key=${PUSHOVER_APP_KEY} --pushover-user-key=${PUSHOVER_USER_KEY} --telegram-app-key=none --telegram-chat-id=none  --ignore-error"
clear

echo -e "${YELLOW}***** Available operations ******"
echo -e "${NC}"
echo ./nefertiti.sh sell
echo ./nefertiti.sh buy market [COIN]
echo ./nefertiti.sh buy signal [SIGNAL]
echo ./nefertiti.sh markets
echo ./nefertiti.sh cancel [COIN]
echo -e "${YELLOW}********************************${NC}"

# MARKETS
if [ "$MODE" == "markets" ]; then
  MARKETS=$(./nefertiti markets --exchange=$EXCHANGE)
  MARKETS=$(echo "${MARKETS}" | tr -s '\n' ' ' | tr -s ',' '\n' | grep '"name"\s*:' | cut -d '"' -f 4)
  for MARKET in ${MARKETS}; do
    if [ "${MARKET: -3}" == "BTC" ]  || [ "${MARKET: -4}" == "USDT" ]; then
      echo ${MARKET}
      ./nefertiti cancel ${ARGS} --market=${MARKET} --side=buy
      ./nefertiti buy ${ARGS} --market=${MARKET} --price=${PRICE} --top=1
      sleep 1
    fi
  done
# MARKETS

elif [ "$MODE" == "profits" ]; then
  echo Nefertiti - PROFITS
  ./nefertiti profits ${ARGS}

# BUYING
elif [ "$MODE" == "buy" ]; then
  # Signal
  if [ "$ARG1" == "signal" ]; then
    if [ "$ARG3" == "" ]; then
      ARG3="BTC"
    fi
    if [ "$ARG2" == "volume" ]; then
      echo Nefertiti - BUY - ${ARG2} signal
      ./nefertiti buy ${ARGS} --quote=${ARG3} --price=${PRICE} --repeat=1 --signals=${ARG2}
    elif [ "$ARG2" == "MiningHamster" ]; then
      echo Nefertiti - BUY - ${ARG2} signal
      ./nefertiti buy ${ARGS} --quote=${ARG3} --price=${PRICE} --repeat=0.005 --signals=${ARG2} --mining-hamster-key=${SIGNAL_MH_KEY}
    elif [ "$ARG2" == "cryptoqualitysignals.com" ]; then
      echo Nefertiti - BUY - ${ARG2} signal
      ./nefertiti buy ${ARGS} --quote=${ARG3} --price=${PRICE} --repeat=0.005 --signals=${ARG2} --quality-signals-key=${SIGNAL_QS_KEY}
    elif [ "$ARG2" == "crypto-tools.net" ]; then
      echo Nefertiti - BUY - ${ARG2} signal
      ./nefertiti buy ${ARGS} --quote=${ARG3} --price=${PRICE} --repeat=0.005 --signals=${ARG2} --crypto-tools-key=${SIGNAL_CT_KEY}
    else ./nefertiti buy ${ARGS} --quote=${ARG3} --price=${PRICE} --repeat=0.005 --signals=crypto-tools.net --crypto-tools-key=${SIGNAL_CT_KEY}
    fi
  # Standard with specific market (XXX/XXX)
  elif [ "$ARG1" == "market" ]; then
    if [ "$ARG3" == "" ]; then
      ARG3="BTC"
    fi
    echo Nefertiti - BUY - ${ARG2}/${ARG3}
    ./nefertiti buy ${ARGS} --market=${ARG2}${ARG3} --price=${PRICE} --repeat=0.0025 --top=1
  fi

# SELLING
elif [ "$MODE" == "sell" ]; then
  echo Nefertiti - SELL - Profit: ${PROFIT}
  ./nefertiti sell ${ARGS} --quote=${QUOTE} --hold=${HOLD} --strategy=1 --mult=${PROFIT} --notify=1

# CANCELLING
elif [ "$MODE" == "cancel" ]; then
  if [ "$ARG3" == "" ]; then
      ARG3="BTC"
  fi
  echo Nefertiti - CANCEL ${ARG1}/${ARG3}
  ./nefertiti cancel ${ARGS} --market=${ARG1}${ARG3} --side=buy
fi