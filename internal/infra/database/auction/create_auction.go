package auction

import (
	"context"
	"fmt"
	"fullcycle-auction_go/configuration/logger"
	"fullcycle-auction_go/internal/entity/auction_entity"
	"fullcycle-auction_go/internal/internal_error"
	"log"
	"os"
	"strconv"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

// Interface Timer para facilitar os testes
type Timer interface {
	After(d time.Duration) <-chan time.Time
}

// Implementação real do Timer
type RealTimer struct{}

func (t *RealTimer) After(d time.Duration) <-chan time.Time {
	return time.After(d)
}

type AuctionEntityMongo struct {
	Id          string                          `bson:"_id"`
	ProductName string                          `bson:"product_name"`
	Category    string                          `bson:"category"`
	Description string                          `bson:"description"`
	Condition   auction_entity.ProductCondition `bson:"condition"`
	Status      auction_entity.AuctionStatus    `bson:"status"`
	Timestamp   int64                           `bson:"timestamp"`
}

type AuctionRepository struct {
	Collection AuctionCollection
	Timer      Timer
	// Funções para facilitar os testes
	InsertOne func(ctx context.Context, document interface{}) (*mongo.InsertOneResult, error)
	UpdateOne func(ctx context.Context, filter interface{}, update interface{}) (*mongo.UpdateResult, error)
}

func NewAuctionRepository(database *mongo.Database) *AuctionRepository {
	coll := &MongoAuctionCollection{collection: database.Collection("auctions")}
	repo := &AuctionRepository{
		Collection: coll,
		Timer:      &RealTimer{},
	}
	repo.InsertOne = func(ctx context.Context, doc interface{}) (*mongo.InsertOneResult, error) {
		return repo.Collection.InsertOne(ctx, doc)
	}
	repo.UpdateOne = func(ctx context.Context, filter interface{}, update interface{}) (*mongo.UpdateResult, error) {
		return repo.Collection.UpdateOne(ctx, filter, update)
	}
	return repo
}

func (ar *AuctionRepository) SetTimer(timer Timer) {
	ar.Timer = timer
}

func (ar *AuctionRepository) CreateAuction(
	ctx context.Context,
	auctionEntity *auction_entity.Auction) *internal_error.InternalError {
	auctionEntityMongo := &AuctionEntityMongo{
		Id:          auctionEntity.Id,
		ProductName: auctionEntity.ProductName,
		Category:    auctionEntity.Category,
		Description: auctionEntity.Description,
		Condition:   auctionEntity.Condition,
		Status:      auctionEntity.Status,
		Timestamp:   auctionEntity.Timestamp.Unix(),
	}

	_, err := ar.InsertOne(ctx, auctionEntityMongo)
	if err != nil {
		logger.Error("Error trying to insert auction", err)
		return internal_error.NewInternalServerError("Error trying to insert auction")
	}

	// Agenda o fechamento automático após a criação bem-sucedida
	go ar.scheduleAuctionClosing(auctionEntityMongo.Id)

	return nil
}
func (ar *AuctionRepository) scheduleAuctionClosing(auctionId string) {
	duration, errDuration := getAuctionDuration()
	if errDuration != nil {
		logger.Error(fmt.Sprintf("Error getting auction duration for auction %s: %v. Auto-closing will not be scheduled.", auctionId, errDuration), errDuration)
		return
	}

	if duration <= 0 {
		log.Printf("Auction duration is %v for auction %s. Auto-closing not scheduled.\n", duration, auctionId)
		return
	}

	log.Printf("Scheduling auction %s to close in %v\n", auctionId, duration)

	// Usa um timer para esperar pela duração
	<-ar.Timer.After(duration) // Bloqueia até o timer disparar

	log.Printf("Timer fired for auction %s. Attempting to close.\n", auctionId)
	// Usa context.Background() ou um contexto mais apropriado se disponível
	closeErr := ar.CloseAuction(context.Background(), auctionId)
	if closeErr != nil {
		logger.Error(fmt.Sprintf("Error automatically closing auction %s", auctionId), closeErr)
	} else {
		log.Printf("Auction %s automatically closed successfully.\n", auctionId)
	}
}

func getAuctionDuration() (time.Duration, *internal_error.InternalError) {
	durationStr := os.Getenv("AUCTION_DURATION_MINUTES")
	if durationStr == "" {
		return 0, internal_error.NewInternalServerError("Variável de ambiente AUCTION_DURATION_MINUTES não definida")
	}

	durationMinutes, err := strconv.Atoi(durationStr)
	if err != nil || durationMinutes <= 0 {
		log.Printf("Formato ou valor inválido para AUCTION_DURATION_MINUTES: %s. Deve ser um inteiro positivo.\n", durationStr)
		return 0, internal_error.NewInternalServerError(fmt.Sprintf("AUCTION_DURATION_MINUTES inválido: %v", err))
	}

	log.Printf("Duração do leilão definida para %d minutos\n", durationMinutes)
	return time.Duration(durationMinutes) * time.Minute, nil
}

func (ar *AuctionRepository) CloseAuction(ctx context.Context, auctionId string) *internal_error.InternalError {
	filter := bson.M{"_id": auctionId, "status": auction_entity.Active} // Fecha apenas se estiver ativo/aberto atualmente
	update := bson.M{"$set": bson.M{
		"status":    auction_entity.Completed, // Define o status como concluído/fechado
		"timestamp": time.Now().Unix(),        // Opcionalmente atualiza o timestamp para o momento do fechamento
	}}

	result, err := ar.Collection.UpdateOne(ctx, filter, update)
	if err != nil {
		logger.Error(fmt.Sprintf("Error updating auction %s to closed status", auctionId), err)
		return internal_error.NewInternalServerError(fmt.Sprintf("Error closing auction: %v", err))
	}

	if result.MatchedCount == 0 {
		log.Printf("Auction %s not found or already closed. No update performed.\n", auctionId)
		// Não é necessariamente um erro, pode já ter sido fechado ou não existir mais.
	} else if result.ModifiedCount == 0 {
		log.Printf("Auction %s matched but not modified (status might already be Completed).\n", auctionId)
	} else {
		log.Printf("Auction %s successfully updated to status Completed.\n", auctionId)
	}
	return nil // Sucesso
}
