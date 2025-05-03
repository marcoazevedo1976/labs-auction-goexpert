package auction_test

import (
	"context"
	"os"
	"testing"
	"time"

	"fullcycle-auction_go/internal/entity/auction_entity"
	"fullcycle-auction_go/internal/infra/database/auction"

	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/mongo"
)

type FakeAuctionCollection struct {
	UpdateCalled chan struct{}
}

func (f *FakeAuctionCollection) InsertOne(ctx context.Context, doc interface{}) (*mongo.InsertOneResult, error) {
	return &mongo.InsertOneResult{}, nil
}

func (f *FakeAuctionCollection) UpdateOne(ctx context.Context, filter interface{}, update interface{}) (*mongo.UpdateResult, error) {
	f.UpdateCalled <- struct{}{}
	return &mongo.UpdateResult{MatchedCount: 1, ModifiedCount: 1}, nil
}
func (f *FakeAuctionCollection) FindOne(ctx context.Context, filter interface{}) auction.SingleResult {
	return nil
}

func (f *FakeAuctionCollection) Find(ctx context.Context, filter interface{}) (auction.Cursor, error) {
	return nil, nil
}

type MockTimer struct {
	C chan time.Time
}

func (m *MockTimer) After(d time.Duration) <-chan time.Time {
	return m.C
}

func TestCreateAuction_ShouldAutoClose(t *testing.T) {
	updateCalled := make(chan struct{}, 1)
	mockCollection := &FakeAuctionCollection{UpdateCalled: updateCalled}
	mockTimer := &MockTimer{C: make(chan time.Time, 1)}

	os.Setenv("AUCTION_DURATION_MINUTES", "1")

	repo := &auction.AuctionRepository{
		Collection: mockCollection,
	}
	repo.SetTimer(mockTimer)
	repo.InsertOne = func(ctx context.Context, doc interface{}) (*mongo.InsertOneResult, error) {
		return mockCollection.InsertOne(ctx, doc)
	}
	repo.UpdateOne = func(ctx context.Context, filter interface{}, update interface{}) (*mongo.UpdateResult, error) {
		return mockCollection.UpdateOne(ctx, filter, update)
	}

	auctionEntity := &auction_entity.Auction{
		Id:          "test123",
		ProductName: "Test Product",
		Category:    "Test Category",
		Description: "Test Description",
		Condition:   auction_entity.New,
		Status:      auction_entity.Active,
		Timestamp:   time.Now(),
	}

	err := repo.CreateAuction(context.Background(), auctionEntity)
	require.Nil(t, err)

	// Dispara o "timer"
	mockTimer.C <- time.Now()

	select {
	case <-updateCalled:
		// OK
	case <-time.After(2 * time.Second):
		t.Fatal("Expected auto-close to be triggered, but it was not")
	}
}
