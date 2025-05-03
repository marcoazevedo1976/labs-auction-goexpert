package auction

import (
	"context"

	"go.mongodb.org/mongo-driver/mongo"
)

type AuctionCollection interface {
	InsertOne(ctx context.Context, document interface{}) (*mongo.InsertOneResult, error)
	UpdateOne(ctx context.Context, filter interface{}, update interface{}) (*mongo.UpdateResult, error)
	FindOne(ctx context.Context, filter interface{}) SingleResult
	Find(ctx context.Context, filter interface{}) (Cursor, error)
}

type SingleResult interface {
	Decode(val interface{}) error
}

type Cursor interface {
	All(ctx context.Context, results interface{}) error
	Close(ctx context.Context) error
}
