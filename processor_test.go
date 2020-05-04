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

package gofer

import (
	"makerdao/gofer/model"
	"makerdao/gofer/query"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"
)


// mockWorkerPool mock worker pool implementation for tests
type mockWorkerPool struct {
	resp *query.HTTPResponse
}

func newMockWorkerPool(resp *query.HTTPResponse) *mockWorkerPool {
	return &mockWorkerPool{
		resp: resp,
	}
}

func (mwp *mockWorkerPool) Ready() bool {
	return true
}

func (mwp *mockWorkerPool) Start() {}

func (mwp *mockWorkerPool) Stop() error {
	return nil
}

func (mwp *mockWorkerPool) Query(req *query.HTTPRequest) *query.HTTPResponse {
	return mwp.resp
}

func newPotentialPricePoint(exchangeName, base, quote string) *model.PotentialPricePoint {
	p := &model.Pair{
		Base:  base,
		Quote: quote,
	}
	return &model.PotentialPricePoint{
		Exchange: &model.Exchange{
			Name: exchangeName,
		},
		Pair: p,
	}
}

func newPricePointFromPotential(pp *model.PotentialPricePoint) *model.PricePoint {
	if err := model.ValidatePotentialPricePoint(pp); err != nil {
		return nil
	}
	return &model.PricePoint{
		Pair: pp.Pair,
		Exchange: pp.Exchange,
		Timestamp: time.Now().Unix(),
		Price: model.PriceFromFloat(1.0),
		Volume: model.PriceFromFloat(1.0),
	}
}

// Define the suite, and absorb the built-in basic suite
// functionality from testify - including a T() method which
// returns the current testing context
type ProcessorSuite struct {
	suite.Suite
}

// All methods that begin with "Test" are run as tests within a
// suite.
func (suite *ProcessorSuite) TestNegativeProcess() {
	pp := newPotentialPricePoint("coinbase", "BTC", "ETH")
	// Wrong worker pool
	p := NewProcessor(nil)
	resp, err := p.Process(pp)
	suite.Nil(resp)
	suite.Error(err)

	p = NewProcessor(newMockWorkerPool(nil))
	resp, err = p.Process(&model.PotentialPricePoint{})
	suite.Nil(resp)
	suite.Error(err)

	wrongPp := newPotentialPricePoint("nonexisting", "BTC", "ETH")
	p = NewProcessor(newMockWorkerPool(nil))
	resp, err = p.Process(wrongPp)
	suite.Nil(resp)
	suite.Error(err)
}

func (suite *ProcessorSuite) TestProcessorSuccess() {
	pp := newPotentialPricePoint("binance", "BTC", "ETH")
	resp := &query.HTTPResponse{
		Body: []byte(`{"price":"1"}`),
	}
	wp := newMockWorkerPool(resp)
	p := NewProcessor(wp)
	point, err := p.Process(pp)

	suite.NoError(err)
	suite.EqualValues(pp.Pair, point.Pair)
	suite.EqualValues(model.PriceFromFloat(1.0), point.Price)
}

// In order for 'go test' to run this suite, we need to create
// a normal test function and pass our suite to suite.Run
func TestProcessorSuite(t *testing.T) {
	suite.Run(t, new(ProcessorSuite))
}
