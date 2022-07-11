package helpers

import (
	"PROJECT_H_server/errors"
	"PROJECT_H_server/global"
	"PROJECT_H_server/schemas"
	Errors "errors"
	"sort"
	"time"

	"github.com/gocql/gocql"
	"github.com/gofiber/fiber/v2"
)

// RelationsMapper get and maps relations data into relations struct
func RelationsMapper(relations *schemas.RelationsSchema, userID string) error {

	iter := global.Session.Query(`
		SELECT * FROM user_relations WHERE user_id = ?;`,
		userID,
	).WithContext(global.Context).Iter()

	defer iter.Close()

	var (
		ok         bool
		relationID gocql.UUID
		curFriend  schemas.FriendsSchema
		curRequest schemas.RequestsSchema
	)
	for {
		row := make(map[string]interface{})
		if !iter.MapScan(row) {
			break
		}
		if relationID, ok = row["relation_id"].(gocql.UUID); ok {

			if row["active"].(bool) {

				if row["friend"].(bool) {

					curFriend.RelationID = relationID.String()
					curFriend.Username = row["relation_username"].(string)
					curFriend.ChainID = row["chain_id"].(gocql.UUID).String()
					curFriend.Created = row["created"].(time.Time).UnixMilli()
					curFriend.LastSeen = row["last_seen"].(time.Time).UnixMilli()
					curFriend.LastRecv = row["last_recv"].(time.Time).UnixMilli()
					relations.Friends = append(relations.Friends, curFriend)

				} else {

					curRequest.RelationID = relationID.String()
					curRequest.Username = row["relation_username"].(string)
					curRequest.Created = row["created"].(time.Time).UnixMilli()
					curRequest.Requested = row["requested"].(bool)
					if curRequest.Requested {
						relations.Requested = append(relations.Requested, curRequest)
					} else {
						relations.Requests = append(relations.Requests, curRequest)
					}

				}

			}

		} else {
			return Errors.New("iter error")
		}
	}

	if len(relations.Requested) >= 1 {
		sort.SliceStable(relations.Requested, func(i, j int) bool {
			return relations.Requested[i].Created < relations.Requested[j].Created
		})
	}

	if len(relations.Requests) >= 1 {
		sort.SliceStable(relations.Requests, func(i, j int) bool {
			return relations.Requests[i].Created > relations.Requests[j].Created
		})
	}

	if len(relations.Friends) >= 1 {
		sort.SliceStable(relations.Friends, func(i, j int) bool {
			return relations.Friends[i].LastRecv > relations.Friends[j].LastRecv
		})
	}

	return nil
}

// CheckExistingRelationChain is a helper function when preparing to request for a new/existing relation
// Returns existing chainID, and error
func CheckExistingRelationChain(c *fiber.Ctx, userID string, relationID string) (gocql.UUID, bool, error) {

	userRelationsResult := make(map[string]interface{})

	err := global.Session.Query(`
		SELECT chain_id FROM user_relations WHERE user_id = ? AND relation_id = ? LIMIT 1;`,
		userID,
		relationID,
	).WithContext(global.Context).MapScan(userRelationsResult)
	if err != nil {
		if err == gocql.ErrNotFound {
			return gocql.UUID{}, false, nil
		}
		return gocql.UUID{}, false, errors.HandleInternalError(c, "users_relations", "ScyllaDB: "+err.Error())
	}

	if userRelationsResult["chain_id"].(gocql.UUID).Time().Equal(time.Time{}) {
		return gocql.UUID{}, false, nil
	}

	return userRelationsResult["chain_id"].(gocql.UUID), true, nil

}
