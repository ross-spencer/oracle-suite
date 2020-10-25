//  Copyright (C) 2020 Maker Ecosystem Growth Holdings, INC.
//
//  This program is free software: you can redistribute it and/or modify
//  it under the terms of the GNU Affero General Public License as
//  published by the Free Software Foundation, either version 3 of the
//  License, or (at your option) any later version.
//
//  This program is distributed in the hope that it will be useful,
//  but WITHOUT ANY WARRANTY; without even the implied warranty of
//  MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
//  GNU Affero General Public License for more details.
//
//  You should have received a copy of the GNU Affero General Public License
//  along with this program.  If not, see <http://www.gnu.org/licenses/>.

package relayer

import (
	"context"
	"errors"
	"math/big"
	"sync"
	"time"

	"github.com/makerdao/gofer/internal/oracle"
	"github.com/makerdao/gofer/pkg/events"
	"github.com/makerdao/gofer/pkg/transport"
)

type Relayer struct {
	mu sync.Mutex

	transport transport.Transport
	interval  time.Duration
	pairs     map[string]Pair
	doneCh    chan bool
}

type Pair struct {
	// AssetPair is the name of asset pair, e.g. ETHUSD.
	AssetPair string
	// OracleSpread is the minimum spread between the oracle price and new price
	// required to send update.
	OracleSpread float64
	// OracleExpiration is the minimum time difference between the oracle time
	// and current time required to send update.
	OracleExpiration time.Duration
	// PriceExpiration is the maximum TTL of the price from feeder.
	PriceExpiration time.Duration
	// Median is the instance of the oracle.Median which is the interface for
	// the median oracle contract.
	Median *oracle.Median
	// prices contains list of prices form the feeders.
	prices *prices
}

func NewRelayer(transport transport.Transport, interval time.Duration) *Relayer {
	return &Relayer{
		transport: transport,
		interval:  interval,
		pairs:     make(map[string]Pair, 0),
		doneCh:    make(chan bool),
	}
}

func (r *Relayer) AddPair(pair Pair) {
	r.mu.Lock()
	defer r.mu.Unlock()

	pair.prices = newPrices()
	r.pairs[pair.AssetPair] = pair
}

func (r *Relayer) Start(successCh chan<- string, errCh chan<- error) error {
	r.startRelayer(successCh, errCh)
	r.startCollector(errCh)

	return nil
}

func (r *Relayer) Stop() {
	r.doneCh <- true
}

func (r *Relayer) collect(price *oracle.Price) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if price.Val.Cmp(big.NewInt(0)) <= 0 {
		return errors.New("invalid price")
	}

	if _, ok := r.pairs[price.AssetPair]; !ok {
		return errors.New("invalid pair")
	}

	err := r.pairs[price.AssetPair].prices.Add(price)
	if err != nil {
		return err
	}

	return nil
}

func (r *Relayer) relay(assetPair string) error {
	ctx := context.Background()
	pair := r.pairs[assetPair]

	oracleQuorum, err := pair.Median.Bar(ctx)
	if err != nil {
		return err
	}
	oracleTime, err := pair.Median.Age(ctx)
	if err != nil {
		return err
	}
	oraclePrice, err := pair.Median.Price(ctx)
	if err != nil {
		return err
	}

	// Clear expired prices:
	pair.prices.ClearOlderThan(time.Now().Add(-1 * pair.PriceExpiration))
	pair.prices.ClearOlderThan(oracleTime)

	// Use only a minimum prices required to achieve a quorum:
	pair.prices.Truncate(oracleQuorum)

	// Check if there are enough prices to achieve a quorum:
	if pair.prices.Len() < oracleQuorum {
		return errors.New("unable to update oracle, there is not enough prices to achieve a quorum")
	}

	// TODO: remove duplicated prices from the same feeder

	isExpired := oracleTime.Add(pair.OracleExpiration).After(time.Now())
	isStale := calcSpread(oraclePrice, pair.prices.Median()) < pair.OracleSpread

	if isExpired || isStale {
		_, err = pair.Median.Poke(ctx, pair.prices.Get())
		pair.prices.Clear()
	}

	return err
}

func (r *Relayer) startCollector(onErrChan chan<- error) {
	go func() {
		for {
			price := &events.Price{}
			select {
			case <-r.doneCh:
				return
			case <-r.transport.WaitFor(price):
				err := r.collect(price.Price)
				if err != nil && onErrChan != nil {
					onErrChan <- err
				}
			}
		}
	}()
}

func (r *Relayer) startRelayer(successCh chan<- string, errCh chan<- error) {
	ticker := time.NewTicker(r.interval)
	go func() {
		for {
			select {
			case <-r.doneCh:
				ticker.Stop()
				return
			case <-ticker.C:
				r.mu.Lock()
				for assetPair, pair := range r.pairs {
					if pair.prices.Len() == 0 {
						continue
					}

					err := r.relay(assetPair)
					if err != nil && errCh != nil {
						errCh <- err
					}
					if err == nil && successCh != nil {
						successCh <- assetPair
					}
				}
				r.mu.Unlock()
			}
		}
	}()
}

func calcSpread(oldPrice, newPrice *big.Int) float64 {
	oldPriceF := new(big.Float).SetInt(oldPrice)
	newPriceF := new(big.Float).SetInt(newPrice)

	x := new(big.Float).Sub(newPriceF, oldPriceF)
	x = new(big.Float).Quo(x, oldPriceF)
	x = new(big.Float).Mul(x, big.NewFloat(100))

	xf, _ := x.Float64()

	return xf
}
