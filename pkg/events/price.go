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

package events

import (
	"encoding/json"

	"github.com/makerdao/gofer/internal/oracle"
)

type Price struct {
	Price *oracle.Price   `json:"price"`
	Trace json.RawMessage `json:"trace"`
}

func (p *Price) Name() string {
	return "price"
}

func (p *Price) PayloadMarshall() ([]byte, error) {
	p.Trace = []byte("null") // remove for now

	// TODO: use binary format and base64 to reduce payload size
	return json.Marshal(p)
}

func (p *Price) PayloadUnmarshall(b []byte) error {
	return json.Unmarshal(b, p)
}
