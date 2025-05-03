package auction

import (
	"context"

	"go.mongodb.org/mongo-driver/mongo"
)

type MongoAuctionCollection struct {
	collection *mongo.Collection
}

func (m *MongoAuctionCollection) InsertOne(ctx context.Context, doc interface{}) (*mongo.InsertOneResult, error) {
	return m.collection.InsertOne(ctx, doc)
}

func (m *MongoAuctionCollection) UpdateOne(ctx context.Context, filter interface{}, update interface{}) (*mongo.UpdateResult, error) {
	return m.collection.UpdateOne(ctx, filter, update)
}

type mongoSingleResult struct {
	sr *mongo.SingleResult
}

func (m *mongoSingleResult) Decode(val interface{}) error {
	return m.sr.Decode(val)
}

type mongoCursor struct {
	cursor *mongo.Cursor
}

func (m *mongoCursor) All(ctx context.Context, results interface{}) error {
	return m.cursor.All(ctx, results)
}

func (m *mongoCursor) Close(ctx context.Context) error {
	return m.cursor.Close(ctx)
}

func (m *MongoAuctionCollection) FindOne(ctx context.Context, filter interface{}) SingleResult {
	return &mongoSingleResult{sr: m.collection.FindOne(ctx, filter)}
}

func (m *MongoAuctionCollection) Find(ctx context.Context, filter interface{}) (Cursor, error) {
	cursor, err := m.collection.Find(ctx, filter)
	if err != nil {
		return nil, err
	}
	return &mongoCursor{cursor: cursor}, nil
}
