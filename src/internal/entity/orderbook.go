package entity

import (
	"errors"
	"fmt"
	"sort"
	"time"

	"github.com/palantir/stacktrace"
)

/*
	Key concept: Limit Order vs Market Order
	https://www.investopedia.com/ask/answers/100314/whats-difference-between-market-order-and-limit-order.asp
*/

var (
	ErrNotFound = errors.New("not found")
)

type OrderType string

const (
	MarketOrder OrderType = "MARKET_ORDER"
	LimitOrder  OrderType = "LIMIT_ORDER"
)

type OrderPlacement string

const (
	BID_ORDER OrderPlacement = "BID"
	ASK_ORDER OrderPlacement = "ASK"
)

type Match struct {
	Ask        *Order
	Bid        *Order
	SizeFilled float64
	Price      float64
}

type Order struct {
	ID             int64          `json:"id"`
	OrderPlacement OrderPlacement `json:"order_placement"`
	Size           float64        `json:"size"`
	Limit          *Limit         `json:"-"`
	Timestamp      int64          `json:"timestamp"`
}

type Orders []*Order

var orderIdSequence int64 = 0

func (o Orders) Len() int           { return len(o) }
func (o Orders) Swap(i, j int)      { o[i], o[j] = o[j], o[i] }
func (o Orders) Less(i, j int) bool { return o[i].Timestamp < o[j].Timestamp }

type OrderMetadata struct {
	Order  *Order
	Market string
}

var OrderIndex = make(map[int64]OrderMetadata)

func NewOrder(orderPlacement OrderPlacement, size float64) *Order {
	orderIdSequence += 1
	return &Order{
		ID:             orderIdSequence,
		Size:           size,
		OrderPlacement: orderPlacement,
		Timestamp:      time.Now().UnixNano(),
	}
}

func (o *Order) IsFilled() bool {
	return o.Size == 0.0
}

func (o *Order) String() string {
	return fmt.Sprintf("[size: %.2f]", o.Size)
}

type Limit struct {
	Price       float64
	Orders      Orders
	TotalVolume float64
}

type Limits []*Limit

type ByBestAsk struct{ Limits }

func (a ByBestAsk) Len() int           { return len(a.Limits) }
func (a ByBestAsk) Swap(i, j int)      { a.Limits[i], a.Limits[j] = a.Limits[j], a.Limits[i] }
func (a ByBestAsk) Less(i, j int) bool { return a.Limits[i].Price < a.Limits[j].Price }

type ByBestBid struct{ Limits }

func (a ByBestBid) Len() int           { return len(a.Limits) }
func (a ByBestBid) Swap(i, j int)      { a.Limits[i], a.Limits[j] = a.Limits[j], a.Limits[i] }
func (a ByBestBid) Less(i, j int) bool { return a.Limits[i].Price > a.Limits[j].Price }

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
	sort.Sort(l.Orders)
}

func (l *Limit) Fill(order *Order) []Match {
	ordersToDelete := []*Order{} // Avoid messing up with order loop

	matches := []Match{}
	for _, matchingOrder := range l.Orders {
		match := l.fillOrder(matchingOrder, order)
		matches = append(matches, match)

		// Remove filled order from limit's entry
		if matchingOrder.IsFilled() {
			ordersToDelete = append(ordersToDelete, matchingOrder)
		}

		// Stop looking for matches
		if order.IsFilled() {
			break
		}
	}

	for _, orderToDelete := range ordersToDelete {
		l.DeleteOrder(orderToDelete)
	}

	return matches
}

func (l *Limit) fillOrder(matchingOrder, order *Order) Match {
	var ask, bid *Order

	if order.OrderPlacement == BID_ORDER {
		ask, bid = matchingOrder, order
	} else {
		ask, bid = order, matchingOrder
	}

	sizeFilled := min(ask.Size, bid.Size)
	ask.Size -= sizeFilled
	bid.Size -= sizeFilled
	l.TotalVolume -= sizeFilled

	return Match{
		Ask:        ask,
		Bid:        bid,
		SizeFilled: sizeFilled,
		Price:      l.Price,
	}
}

func (l *Limit) String() string {
	return fmt.Sprintf("[price: %.2f | volume: %.2f]", l.Price, l.TotalVolume)
}

type OrderBook struct {
	Market string

	asks []*Limit
	bids []*Limit

	AskLimits map[float64]*Limit
	BidLimits map[float64]*Limit
}

func NewOrderBook(market string) *OrderBook {
	return &OrderBook{
		Market:    market,
		asks:      []*Limit{},
		bids:      []*Limit{},
		AskLimits: make(map[float64]*Limit),
		BidLimits: make(map[float64]*Limit),
	}
}

func (ob *OrderBook) PlaceMarketOrder(order *Order) ([]Match, error) {
	matches := []Match{}

	if order.OrderPlacement == BID_ORDER {
		if order.Size > ob.AskTotalVolume() {
			return nil, stacktrace.NewError("PlaceMarketOrder: not enough ask volume in the market. asks: %.2f, bids: %.2f", ob.AskTotalVolume(), order.Size)
		}
		for _, limit := range ob.Asks() {
			limitMatches := limit.Fill(order)
			if limit.TotalVolume == 0 {
				ob.deleteLimit(ASK_ORDER, limit)
			}
			matches = append(matches, limitMatches...)
			if order.IsFilled() {
				break
			}
		}
	} else {
		if order.Size > ob.BidTotalVolume() {
			return nil, stacktrace.NewError("PlaceMarketOrder: not enough bid volume in the market. asks: %.2f, bids: %.2f", order.Size, ob.BidTotalVolume())
		}
		for _, limit := range ob.Bids() {
			limitMatches := limit.Fill(order)
			if limit.TotalVolume == 0 {
				ob.deleteLimit(BID_ORDER, limit)
			}
			matches = append(matches, limitMatches...)
			if order.IsFilled() {
				break
			}
		}
	}

	return matches, nil
}

func (ob *OrderBook) AskTotalVolume() float64 {
	totalVolume := 0.0
	for _, ask := range ob.asks {
		totalVolume += ask.TotalVolume
	}

	return totalVolume
}

func (ob *OrderBook) BidTotalVolume() float64 {
	totalVolume := 0.0
	for _, bid := range ob.bids {
		totalVolume += bid.TotalVolume
	}

	return totalVolume
}

func (ob *OrderBook) PlaceLimitOrder(price float64, order *Order) error {
	var limit *Limit
	if order.OrderPlacement == BID_ORDER {
		limit = ob.BidLimits[price]
	} else if order.OrderPlacement == ASK_ORDER {
		limit = ob.AskLimits[price]
	} else {
		return errors.New("invalid order placement")
	}

	// Limit volume doesn't exist yet
	if limit == nil {
		limit = NewLimit(price)
		if order.OrderPlacement == BID_ORDER {
			ob.bids = append(ob.bids, limit)
			ob.BidLimits[price] = limit
		} else {
			ob.asks = append(ob.asks, limit)
			ob.AskLimits[price] = limit
		}
	}

	limit.AddOrder(order)
	OrderIndex[order.ID] = OrderMetadata{
		Order:  order,
		Market: ob.Market,
	}

	return nil
}

func (ob *OrderBook) CancelOrderByID(orderId int64, orderPlacement OrderPlacement) error {
	var limits []*Limit
	if orderPlacement == BID_ORDER {
		limits = ob.bids
	} else {
		limits = ob.asks
	}

	for _, limit := range limits {
		for _, order := range limit.Orders {
			if order.ID == orderId {
				limit.DeleteOrder(order)
				delete(OrderIndex, order.ID)
				if limit.TotalVolume == 0 {
					ob.deleteLimit(orderPlacement, limit)
				}
				return nil
			}
		}
	}

	return ErrNotFound
}

func (ob *OrderBook) deleteLimit(limitPlacement OrderPlacement, limit *Limit) {
	if limitPlacement == BID_ORDER {
		delete(ob.BidLimits, limit.Price)

		for i := 0; i < len(ob.bids); i++ {
			if ob.bids[i] == limit {
				ob.bids[i] = ob.bids[len(ob.bids)-1]
				ob.bids = ob.bids[:len(ob.bids)-1]
			}
		}
	} else {
		delete(ob.AskLimits, limit.Price)

		for i := 0; i < len(ob.asks); i++ {
			if ob.asks[i] == limit {
				ob.asks[i] = ob.asks[len(ob.asks)-1]
				ob.asks = ob.asks[:len(ob.asks)-1]
			}
		}
	}
}

func (ob *OrderBook) Asks() []*Limit {
	sort.Sort(ByBestAsk{ob.asks})
	return ob.asks
}

func (ob *OrderBook) Bids() []*Limit {
	sort.Sort(ByBestBid{ob.bids})
	return ob.bids
}
