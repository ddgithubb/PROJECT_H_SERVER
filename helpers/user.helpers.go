package helpers

import (
	"PROJECT_H_server/errors"
	"PROJECT_H_server/global"
	"PROJECT_H_server/schemas"
	Errors "errors"
	"time"

	"github.com/gocql/gocql"
	"github.com/gofiber/fiber/v2"
)

// GetUsernameByID gets only the username column by id
func GetUsernameByID(c *fiber.Ctx, id string) (string, error) {
	reqUsername := ""

	err := global.Session.Query(`
		SELECT username FROM users_private WHERE user_id = ? LIMIT 1;`,
		id,
	).WithContext(global.Context).Scan(&reqUsername)

	if err != nil {
		if err == gocql.ErrNotFound {
			return "", errors.HandleBadRequestError(c, "UserID", "invalid")
		}
		return "", errors.HandleInternalError(c, "users_private", "ScyllaDB: "+err.Error())
	}

	return reqUsername, err
}

// CheckUser checks user by id
func CheckUser(id string) (bool, error) {
	var existCount int

	err := global.Session.Query(`
		SELECT count(*) FROM users_private WHERE user_id = ? LIMIT 1;`,
		id,
	).WithContext(global.Context).Scan(&existCount)

	if existCount == 0 {
		return false, err
	}
	return true, err
}

// GetUserInfo gets user info on authentication
func GetUserInfo(c *fiber.Ctx, userID string) (schemas.UserInfoSchema, error) {
	userResult := make(map[string]interface{})

	err := global.Session.Query(`
		SELECT * FROM users_private WHERE user_id = ? LIMIT 1;`,
		userID,
	).WithContext(global.Context).MapScan(userResult)

	if err != nil {
		if err == gocql.ErrNotFound {
			return schemas.UserInfoSchema{UserID: ""}, errors.HandleInternalError(c, "users_private", "ScyllaDB: "+err.Error())
		}
		return schemas.UserInfoSchema{UserID: ""}, errors.HandleInternalError(c, "users_private", "ScyllaDB: "+err.Error())
	}

	initUserInfo := schemas.UserInfoSchema{
		UserID:   userID,
		Username: userResult["username"].(string),
	}

	initUserInfo.Relations, err = RelationsMapper(userID)
	if err != nil {
		return schemas.UserInfoSchema{UserID: ""}, errors.HandleInternalError(c, "user_relations", err.Error())
	}

	if statement, ok := userResult["statement"].(string); ok {
		initUserInfo.Statement = statement
	}

	return initUserInfo, nil
}

// RelationsMapper get and maps relations data into relations struct
func RelationsMapper(userID string) (schemas.RelationsSchema, error) {

	iter := global.Session.Query(`
		SELECT * FROM user_relations WHERE user_id = ?;`,
		userID,
	).WithContext(global.Context).Iter()

	var (
		ok         bool
		relationID gocql.UUID
		relations  schemas.RelationsSchema
		curFriend  schemas.FriendsSchema
		curRequest schemas.RequestsSchema
	)
	for {
		row := make(map[string]interface{})
		if !iter.MapScan(row) {
			break
		}
		if relationID, ok = row["relation_id"].(gocql.UUID); ok {
			if row["friend"].(bool) {
				curFriend.RelationID = relationID.String()
				curFriend.Username = row["relation_username"].(string)
				curFriend.ChainID = row["chain_id"].(gocql.UUID).String()
				curFriend.LastSeen = row["last_seen"].(time.Time).UnixMilli()
				curFriend.LastRecv = row["last_recv"].(time.Time).UnixMilli()
				relations.Friends = append([]schemas.FriendsSchema{curFriend}, relations.Friends...)
			} else {
				curRequest.RelationID = relationID.String()
				curRequest.Username = row["relation_username"].(string)
				curRequest.Requested = row["requested"].(bool)
				if curRequest.Requested {
					relations.Requested = append([]schemas.RequestsSchema{curRequest}, relations.Requested...)
				} else {
					relations.Requests = append(relations.Requests, curRequest)
				}
			}
		} else {
			return schemas.RelationsSchema{}, Errors.New("iter error")
		}
	}
	for i := 0; i < len(relations.Friends); i++ {
		relations.Friends[i].Key = i
	}
	// if len(relations.Friends) >= 1 {
	// 	sort.SliceStable(relations.Friends, func(i, j int) bool {
	// 		return relations.Friends[i].LastRecv > relations.Friends[j].LastRecv
	// 	})
	// }
	return relations, nil
}
