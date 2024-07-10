package entity_test

import (
	"fmt"
	"testing"

	"github.com/idzharbae/crypto-exchange/src/internal/entity"
)

func TestLimit(t *testing.T) {
	l := entity.NewLimit(10_000)
	buyOrder := entity.NewOrder(entity.BID_ORDER, 5)
	l.AddOrder(buyOrder)
	sellOrder := entity.NewOrder(entity.ASK_ORDER, 3)
	l.AddOrder(sellOrder)
	sellOrder2 := entity.NewOrder(entity.ASK_ORDER, 13)
	l.AddOrder(sellOrder2)
	sellOrder3 := entity.NewOrder(entity.ASK_ORDER, 7)
	l.AddOrder(sellOrder3)

	l.DeleteOrder(sellOrder2)

	fmt.Println(l)
}

func TestOrderBook(t *testing.T) {
	ob := entity.NewOrderBook()

	buyOrder := entity.NewOrder(entity.BID_ORDER, 10)
	buyOrder2 := entity.NewOrder(entity.BID_ORDER, 2000)
	ob.PlaceOrder(18_000, buyOrder)
	ob.PlaceOrder(18_000, buyOrder2)

	fmt.Println(ob.Bids)
}
