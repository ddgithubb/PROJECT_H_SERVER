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

// GetChain gets a limited amount of chain based on create
func GetChain(chainID gocql.UUID, reqTime time.Time, asc bool, new bool, limit int64) ([]schemas.MessageSchema, error) {

	var iter *gocql.Iter

	if limit > 10 {
		limit = 10
	} else if limit <= 0 {
		return []schemas.MessageSchema{}, nil
	}

	if !new {
		if asc {
			iter = global.Session.Query(`
				SELECT * FROM chains WHERE chain_id = ? AND created > ? ORDER BY created ASC LIMIT ? BYPASS CACHE;`,
				chainID,
				reqTime,
				limit,
			).WithContext(global.Context).Iter()
		} else {
			iter = global.Session.Query(`
				SELECT * FROM chains WHERE chain_id = ? AND created < ? LIMIT ? BYPASS CACHE;`,
				chainID,
				reqTime,
				limit,
			).WithContext(global.Context).Iter()
		}
	} else {
		iter = global.Session.Query(`
			SELECT * FROM chains WHERE chain_id = ? AND created <= ? LIMIT ? BYPASS CACHE;`,
			chainID,
			reqTime,
			limit,
		).WithContext(global.Context).Iter()
	}

	defer iter.Close()
	chain := []schemas.MessageSchema{}

	var (
		messageID  gocql.UUID
		ok         bool
		curMessage schemas.MessageSchema
	)
	for {
		row := make(map[string]interface{})
		if !iter.MapScan(row) {
			break
		}

		if messageID, ok = row["message_id"].(gocql.UUID); ok {
			curMessage.MessageID = messageID.String()
			curMessage.UserID = row["user_id"].(gocql.UUID).String()
			curMessage.Created = messageID.Time().UnixMilli()
			curMessage.Expires = row["expires"].(time.Time).UnixMilli()
			curMessage.Type = row["type"].(int)
			curMessage.Seen = row["seen"].(bool)
			curMessage.Display = row["display"].(string)
			curMessage.Duration = row["duration"].(int)
			chain = append(chain, curMessage)
		} else {
			return nil, Errors.New("iter error")
		}
	}

	if !asc {
		for i := 0; i < len(chain)/2; i++ {
			j := len(chain) - i - 1
			chain[i], chain[j] = chain[j], chain[i]
		}
	}

	return chain, nil
}

func InsertChainsUsers(c *fiber.Ctx, chainID gocql.UUID, users ...string) error {

	created := time.Now().UTC()

	b := global.Session.NewBatch(gocql.LoggedBatch).WithContext(global.Context)

	for i := 0; i < len(users); i++ {
		b.Entries = append(b.Entries, gocql.BatchEntry{
			Stmt:       "INSERT INTO chains_users (chain_id, user_id, created) VALUES (?, ?, ?)",
			Args:       []interface{}{chainID, users[i], created},
			Idempotent: true,
		})
	}

	err := global.Session.ExecuteBatch(b)
	if err != nil {
		return errors.HandleInternalError(c, "chains_users", "ScyllaDB: "+err.Error())
	}

	return nil

}

func GetChainsUsers(c *fiber.Ctx, chainID gocql.UUID) ([]string, error) {

	iter := global.Session.Query(`
		SELECT * FROM chains_users WHERE chain_id = ?;`,
		chainID,
	).WithContext(global.Context).Iter()

	defer iter.Close()

	users := []string{}

	var (
		ok     bool
		userID gocql.UUID
	)
	for {
		row := make(map[string]interface{})
		if !iter.MapScan(row) {
			break
		}
		if userID, ok = row["user_id"].(gocql.UUID); ok {
			users = append(users, userID.String())
		} else {
			return nil, Errors.New("iter error")
		}
	}

	return users, nil

}

// ParseParamChainUUID parses chain uuid from parameter
func ParseParamChainUUID(c *fiber.Ctx) (gocql.UUID, error) {
	chainID, err := gocql.ParseUUID(c.Params("chainID"))
	if err != nil {
		return gocql.UUID{}, errors.HandleBadRequestError(c, "ChainID", "invalid")
	}

	return chainID, err
}

// ParseParamMessageUUID parses message uuid from parameter
func ParseParamMessageUUID(c *fiber.Ctx) (gocql.UUID, error) {
	messageID, err := gocql.ParseUUID(c.Params("messageID"))
	if err != nil {
		return gocql.UUID{}, errors.HandleBadRequestError(c, "MessageID", "invalid")
	}
	return messageID, err
}
