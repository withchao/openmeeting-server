package mgo

import (
	"context"
	"github.com/openimsdk/openmeeting-server/pkg/common/storage/database"
	"github.com/openimsdk/openmeeting-server/pkg/common/storage/model"
	"github.com/openimsdk/tools/db/mongoutil"
	"github.com/openimsdk/tools/db/pagination"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"time"
)

func NewSignal(db *mongo.Database) (database.SignalInterface, error) {
	coll := db.Collection("signal")
	_, err := coll.Indexes().CreateMany(context.Background(), []mongo.IndexModel{
		{
			Keys: bson.D{
				{Key: "sid", Value: 1},
			},
			Options: options.Index().SetUnique(true),
		},
		{
			Keys: bson.D{
				{Key: "inviter_user_id", Value: 1},
			},
		},
		{
			Keys: bson.D{
				{Key: "initiate_time", Value: -1},
			},
		},
	})
	if err != nil {
		return nil, err
	}
	return &signal{coll: coll}, nil
}

type signal struct {
	coll *mongo.Collection
}

func (x *signal) Find(ctx context.Context, sids []string) ([]*model.SignalModel, error) {
	return mongoutil.Find[*model.SignalModel](ctx, x.coll, bson.M{"sid": bson.M{"$in": sids}})
}

func (x *signal) CreateSignal(ctx context.Context, signalModel *model.SignalModel) error {
	return mongoutil.InsertMany(ctx, x.coll, []*model.SignalModel{signalModel})
}

func (x *signal) Update(ctx context.Context, sid string, update map[string]any) error {
	if len(update) == 0 {
		return nil
	}
	return mongoutil.UpdateOne(ctx, x.coll, bson.M{"sid": sid}, bson.M{"$set": update}, false)
}

func (x *signal) UpdateSignalFileURL(ctx context.Context, sID, fileURL string) error {
	return x.Update(ctx, sID, map[string]any{"file_url": fileURL})
}

func (x *signal) UpdateSignalEndTime(ctx context.Context, sID string, endTime time.Time) error {
	return x.Update(ctx, sID, map[string]any{"end_time": endTime})
}

func (x *signal) Delete(ctx context.Context, sids []string) error {
	if len(sids) == 0 {
		return nil
	}
	return mongoutil.DeleteMany(ctx, x.coll, bson.M{"sid": bson.M{"$in": sids}})
}

func (x *signal) PageSignal(ctx context.Context, sesstionType int32, sendID string, startTime, endTime time.Time, pagination pagination.Pagination) (int64, []*model.SignalModel, error) {
	var and []bson.M
	if !startTime.IsZero() {
		and = append(and, bson.M{"initiate_time": bson.M{"$gte": startTime}})
	}
	if !endTime.IsZero() {
		and = append(and, bson.M{"initiate_time": bson.M{"$lte": endTime}})
	}
	if sesstionType != 0 {
		and = append(and, bson.M{"session_type": sesstionType})
	}
	if sendID != "" {
		and = append(and, bson.M{"inviter_user_id": sendID})
	}
	var filter any
	if len(and) == 0 {
		filter = bson.M{}
	} else {
		filter = bson.M{"$and": and}
	}
	return mongoutil.FindPage[*model.SignalModel](ctx, x.coll, filter, pagination, options.Find().SetSort(bson.M{"initiate_time": -1}))
}
