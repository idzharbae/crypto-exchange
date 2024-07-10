package entity_test

import (
	"fmt"
	"testing"

	"github.com/idzharbae/crypto-exchange/src/internal/entity"
	. "github.com/smartystreets/goconvey/convey"
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

func TestPlaceLimitOrder(t *testing.T) {
	Convey("Test PlaceLimitOrder", t, func() {
		ob := entity.NewOrderBook()

		buyOrder := entity.NewOrder(entity.BID_ORDER, 10)
		buyOrder2 := entity.NewOrder(entity.BID_ORDER, 2000)
		ob.PlaceLimitOrder(10_000, buyOrder)
		ob.PlaceLimitOrder(18_000, buyOrder2)

		So(len(ob.Bids()), ShouldEqual, 2)
	})
}

func TestPlaceMarketOrder(t *testing.T) {
	Convey("When placing market order", t, func() {
		Convey("Should return error if not enough volume", func() {
			ob := entity.NewOrderBook()
			sellOrder := entity.NewOrder(entity.ASK_ORDER, 100)
			ob.PlaceLimitOrder(10_000, sellOrder)
			sellOrder2 := entity.NewOrder(entity.ASK_ORDER, 100)
			ob.PlaceLimitOrder(12_000, sellOrder2)

			buyOrder := entity.NewOrder(entity.BID_ORDER, 500)
			_, err := ob.PlaceMarketOrder(buyOrder)

			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, "PlaceMarketOrder: not enough ask volume in the market. asks: 200.00, bids: 500.00")
		})

		Convey("Should return error if not enough volume (ask)", func() {
			ob := entity.NewOrderBook()
			buyOrder := entity.NewOrder(entity.BID_ORDER, 100)
			ob.PlaceLimitOrder(10_000, buyOrder)
			buyOrder2 := entity.NewOrder(entity.BID_ORDER, 100)
			ob.PlaceLimitOrder(12_000, buyOrder2)

			sellOrder := entity.NewOrder(entity.ASK_ORDER, 500)
			_, err := ob.PlaceMarketOrder(sellOrder)

			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, "PlaceMarketOrder: not enough bid volume in the market. asks: 500.00, bids: 200.00")
		})

		Convey("Should return matches if volume is enough", func() {
			ob := entity.NewOrderBook()
			sellOrder := entity.NewOrder(entity.ASK_ORDER, 100)
			ob.PlaceLimitOrder(10_000, sellOrder)
			sellOrder2 := entity.NewOrder(entity.ASK_ORDER, 350)
			ob.PlaceLimitOrder(12_000, sellOrder2)
			sellOrder3 := entity.NewOrder(entity.ASK_ORDER, 15)
			ob.PlaceLimitOrder(13_000, sellOrder3)
			sellOrder4 := entity.NewOrder(entity.ASK_ORDER, 51)
			ob.PlaceLimitOrder(12_000, sellOrder4)

			buyOrder := entity.NewOrder(entity.BID_ORDER, 500)
			matches, err := ob.PlaceMarketOrder(buyOrder)

			So(err, ShouldBeNil)
			So(len(matches), ShouldEqual, 3)
			So(ob.AskTotalVolume(), ShouldEqual, 16) // Should reduce volume count
			So(matches[0].Ask, ShouldEqual, sellOrder)
			So(matches[0].Bid, ShouldEqual, buyOrder)
			So(matches[1].Ask, ShouldEqual, sellOrder2)
			So(matches[1].Bid, ShouldEqual, buyOrder)
			So(matches[2].Ask, ShouldEqual, sellOrder4)
			So(matches[2].Bid, ShouldEqual, buyOrder)
			So(sellOrder4.Size, ShouldEqual, 1)
			So(len(ob.Asks()), ShouldEqual, 2)
		})

		Convey("Should return matches if volume is enough (ask)", func() {
			ob := entity.NewOrderBook()
			buyOrder := entity.NewOrder(entity.BID_ORDER, 100)
			ob.PlaceLimitOrder(13_000, buyOrder)
			buyOrder2 := entity.NewOrder(entity.BID_ORDER, 350)
			ob.PlaceLimitOrder(12_000, buyOrder2)
			buyOrder3 := entity.NewOrder(entity.BID_ORDER, 15)
			ob.PlaceLimitOrder(10_000, buyOrder3)
			buyOrder4 := entity.NewOrder(entity.BID_ORDER, 51)
			ob.PlaceLimitOrder(12_000, buyOrder4)

			sellOrder := entity.NewOrder(entity.ASK_ORDER, 500)
			matches, err := ob.PlaceMarketOrder(sellOrder)

			So(err, ShouldBeNil)
			So(len(matches), ShouldEqual, 3)
			So(ob.BidTotalVolume(), ShouldEqual, 16) // Should reduce volume count
			So(matches[0].Bid, ShouldEqual, buyOrder)
			So(matches[0].Ask, ShouldEqual, sellOrder)
			So(matches[1].Bid, ShouldEqual, buyOrder2)
			So(matches[1].Ask, ShouldEqual, sellOrder)
			So(matches[2].Bid, ShouldEqual, buyOrder4)
			So(matches[2].Ask, ShouldEqual, sellOrder)
			So(buyOrder4.Size, ShouldEqual, 1)
			So(len(ob.Bids()), ShouldEqual, 2)
		})
	})
}
