package mgo

import (
	"context"
	"github.com/openimsdk/openmeeting-server/pkg/common/storage/database"
	"github.com/openimsdk/openmeeting-server/pkg/common/storage/model"
	"github.com/openimsdk/tools/db/mongoutil"
	"github.com/openimsdk/tools/db/pagination"
	"github.com/openimsdk/tools/errs"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"time"
)

func NewSignalInvitation(db *mongo.Database) (database.SignalInvitationInterface, error) {
	coll := db.Collection("signal_invitation")
	_, err := coll.Indexes().CreateMany(context.Background(), []mongo.IndexModel{
		{
			Keys: bson.D{
				{Key: "sid", Value: 1},
				{Key: "user_id", Value: 1},
			},
			Options: options.Index().SetUnique(true),
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
	return &signalInvitation{coll: coll}, nil
}

type signalInvitation struct {
	coll *mongo.Collection
}

func (x *signalInvitation) Find(ctx context.Context, sid string) ([]*model.SignalInvitationModel, error) {
	return mongoutil.Find[*model.SignalInvitationModel](ctx, x.coll, bson.M{"sid": sid})
}

func (x *signalInvitation) UpdateMany(ctx context.Context, sid string, inviteeUserIDs []string) error {
	var operations []mongo.WriteModel
	now := time.Now()
	for _, userID := range inviteeUserIDs {
		if userID == "" {
			continue
		}
		filter := bson.M{"sid": sid, "user_id": userID}
		update := bson.M{
			"$set": bson.M{
				"user_id":       userID,
				"sid":           sid,
				"initiate_time": now,
				"handle_time":   time.Unix(0, 0),
			},
		}

		operations = append(operations, mongo.NewUpdateOneModel().
			SetFilter(filter).
			SetUpdate(update).
			SetUpsert(true))
	}
	if len(operations) == 0 {
		return nil
	}
	bulkOptions := options.BulkWrite().SetOrdered(false)
	_, err := x.coll.BulkWrite(ctx, operations, bulkOptions)
	return err
}

func (x *signalInvitation) CreateSignalInvitation(ctx context.Context, sid string, inviteeUserIDs []string) error {
	if err := x.UpdateMany(ctx, sid, inviteeUserIDs); err != nil {
		return errs.WrapMsg(err, "mongo update many")
	}
	return nil
	/*now := time.Now()
		return mongoutil.InsertMany(ctx, x.coll, datautil.Slice(inviteeUserIDs, func(userID string) *model.SignalInvitationModel {
		return &model.SignalInvitationModel{
			UserID:       userID,
			SID:          sid,
			InitiateTime: now,
			HandleTime:   time.Unix(0, 0),
		}
	}))*/
}

func (x *signalInvitation) HandleSignalInvitation(ctx context.Context, sID, InviteeUserID string, status int32) error {
	return mongoutil.UpdateOne(ctx, x.coll, bson.M{"sid": sID, "user_id": InviteeUserID}, bson.M{"$set": bson.M{"status": status, "handle_time": time.Now()}}, false)
}

func (x *signalInvitation) PageSID(ctx context.Context, recvID string, startTime, endTime time.Time, pagination pagination.Pagination) (int64, []string, error) {
	var and []bson.M
	and = append(and, bson.M{"user_id": recvID})
	if !startTime.IsZero() {
		and = append(and, bson.M{"initiate_time": bson.M{"$gte": startTime}})
	}
	if !endTime.IsZero() {
		and = append(and, bson.M{"initiate_time": bson.M{"$lte": endTime}})
	}
	return mongoutil.FindPage[string](ctx, x.coll, bson.M{"$and": and}, pagination, options.Find().SetProjection(bson.M{"_id": 0, "sid": 1}).SetSort(bson.M{"initiate_time": -1}))
}

func (x *signalInvitation) Delete(ctx context.Context, sids []string) error {
	if len(sids) == 0 {
		return nil
	}
	return mongoutil.DeleteMany(ctx, x.coll, bson.M{"sid": bson.M{"$in": sids}})
}