package entity

import (
	"fmt"
	"time"
)

/*
	Key concept: Limit Order vs Market Order
	https://www.investopedia.com/ask/answers/100314/whats-difference-between-market-order-and-limit-order.asp
*/

type OrderType int

const (
	BID_ORDER OrderType = iota // BUY
	ASK_ORDER                  // SELL
)

type Match struct {
	Ask        *Order
	Bid        *Order
	SizeFilled float64
	Price      float64
}

type Order struct {
	Type      OrderType
	Size      float64
	Limit     *Limit
	Timestamp int64
}

func NewOrder(orderType OrderType, size float64) *Order {
	return &Order{
		Size:      size,
		Type:      orderType,
		Timestamp: time.Now().UnixNano(),
	}
}

func (o *Order) String() string {
	return fmt.Sprintf("[size: %.2f]", o.Size)
}

type Limit struct {
	Price       float64
	Orders      []*Order
	TotalVolume float64
}

func NewLimit(price float64) *Limit {
	return &Limit{
		Price:  price,
		Orders: []*Order{},
	}
}

func (l *Limit) AddOrder(o *Order) {
	o.Limit = l
	l.Orders = append(l.Orders, o)
	l.TotalVolume += o.Size
}

func (l *Limit) DeleteOrder(o *Order) {
	for i := 0; i < len(l.Orders); i++ {
		if l.Orders[i] == o {
			// Copy last element to current index to overwrite it and then trim the last element to free up memory
			l.Orders[i] = l.Orders[len(l.Orders)-1]
			l.Orders = l.Orders[:len(l.Orders)-1]
			break
		}
	}

	o.Limit = nil
	l.TotalVolume -= o.Size
}

func (l *Limit) String() string {
	return fmt.Sprintf("[price: %.2f | volume: %.2f]", l.Price, l.TotalVolume)
}

type OrderBook struct {
	Asks []*Limit
	Bids []*Limit

	AskLimits map[float64]*Limit
	BidLimits map[float64]*Limit
}

func NewOrderBook() *OrderBook {
	return &OrderBook{
		Asks:      []*Limit{},
		Bids:      []*Limit{},
		AskLimits: make(map[float64]*Limit),
		BidLimits: make(map[float64]*Limit),
	}
}

func (ob *OrderBook) PlaceOrder(price float64, order *Order) []Match {
	// TODO: Find matching order

	// If still have left over order size, place it on the order book
	if order.Size > 0.0 {
		ob.add(price, order)
	}

	return []Match{}
}

func (ob *OrderBook) add(price float64, order *Order) {
	var limit *Limit
	if order.Type == BID_ORDER {
		limit = ob.BidLimits[price]
	} else {
		limit = ob.AskLimits[price]
	}

	// Limit volume doesn't exist yet
	if limit == nil {
		limit = NewLimit(price)
		if order.Type == BID_ORDER {
			ob.Bids = append(ob.Bids, limit)
			ob.BidLimits[price] = limit
		} else {
			ob.Asks = append(ob.Asks, limit)
			ob.AskLimits[price] = limit
		}
	}

	limit.AddOrder(order)
}
