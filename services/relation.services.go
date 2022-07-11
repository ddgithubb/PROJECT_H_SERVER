package services

import (
	"PROJECT_H_server/errors"
	"PROJECT_H_server/global"
	"PROJECT_H_server/helpers"
	"PROJECT_H_server/messages"
	"PROJECT_H_server/schemas"
	"time"

	"github.com/gocql/gocql"
	"github.com/gofiber/fiber/v2"
)

func GetAllRelations(c *fiber.Ctx) error {

	relations := new(schemas.RelationsSchema)
	err := helpers.RelationsMapper(relations, c.Locals("userid").(string))
	if err != nil {
		return errors.HandleInternalError(c, "user_relations", err.Error())
	}

	return c.JSON(relations)
}

// RequestRelation requests selected user by id
func RequestRelation(c *fiber.Ctx) error {

	userID := c.Locals("userid").(string)
	username := c.Locals("username").(string)
	relationID := c.Params("relationID")

	relationUsername, err := helpers.GetUsernameByID(c, relationID)
	if err != nil {
		return err
	}

	created := time.Now().UTC()

	err = global.Session.Query(`
		UPDATE user_relations 
		SET 
		created = ?,
		chain_id = ?,
		relation_username = ?,
		last_recv = ?,
		last_seen = ?,
		friend = ?,
		requested = ?,
		active = ? 
		WHERE user_id = ? AND relation_id = ?;`,
		created,
		gocql.UnsetValue,
		relationUsername,
		gocql.UnsetValue,
		gocql.UnsetValue,
		gocql.UnsetValue,
		true,
		true,
		userID,
		relationID,
	).WithContext(global.Context).Exec()

	if err != nil {
		return errors.HandleInternalError(c, "user_relations", "ScyllaDB: "+err.Error())
	}

	err = global.Session.Query(`
		UPDATE user_relations 
		SET 
		created = ?,
		chain_id = ?,
		relation_username = ?,
		last_recv = ?,
		last_seen = ?,
		friend = ?,
		requested = ?,
		active = ? 
		WHERE user_id = ? AND relation_id = ?;`,
		created,
		gocql.UnsetValue,
		username,
		gocql.UnsetValue,
		gocql.UnsetValue,
		gocql.UnsetValue,
		false,
		true,
		relationID,
		userID,
	).WithContext(global.Context).Exec()

	if err != nil {
		return errors.HandleInternalError(c, "user_relations", "ScyllaDB: "+err.Error())
	}

	if err = messages.FriendRequest(c, relationID, relationUsername); err != nil {
		return err
	}

	return c.JSON(struct {
		Username string
	}{
		Username: relationUsername,
	})
}

// AcceptRelation accepts request from user by id
func AcceptRelation(c *fiber.Ctx) error {

	userID := c.Locals("userid").(string)
	relationID := c.Params("relationID")

	chainID, valid, err := helpers.CheckExistingRelationChain(c, userID, relationID)
	if err != nil {
		return err
	}

	if !valid {
		chainID = gocql.TimeUUID()

		if err = helpers.InsertChainsUsers(c, chainID, userID, relationID); err != nil {
			return err
		}
	}

	curTime := time.Now().UTC()

	err = global.Session.Query(`
		UPDATE user_relations 
		SET 
		chain_id = ?,
		last_recv = ?,
		last_seen = ?,
		friend = ? 
		WHERE user_id = ? AND relation_id = ?;`,
		chainID,
		curTime,
		curTime,
		true,
		userID,
		relationID,
	).WithContext(global.Context).Exec()

	if err != nil {
		return errors.HandleInternalError(c, "user_relations", "ScyllaDB: "+err.Error())
	}

	err = global.Session.Query(`
		UPDATE user_relations 
		SET 
		chain_id = ?,
		last_recv = ?,
		last_seen = ?,
		friend = ? 
		WHERE user_id = ? AND relation_id = ?;`,
		chainID,
		curTime,
		curTime,
		true,
		relationID,
		userID,
	).WithContext(global.Context).Exec()

	if err != nil {
		return errors.HandleInternalError(c, "user_relations", "ScyllaDB: "+err.Error())
	}

	if err = messages.FriendAccepted(c, relationID, chainID.String(), curTime.UnixMilli()); err != nil {
		return err
	}

	return c.JSON(struct {
		ChainID string
		Updated int64
	}{
		ChainID: chainID.String(),
		Updated: curTime.UnixMilli(),
	})
}

// RemoveRelations removes relation selected user by id
func RemoveRelation(c *fiber.Ctx) error {

	userID := c.Locals("userid").(string)
	relationID := c.Params("relationID")

	err := global.Session.Query(`
		UPDATE user_relations 
		SET 
		friend = ?,
		requested = ?,
		active = ? 
		WHERE user_id = ? AND relation_id = ?;`,
		false,
		false,
		false,
		userID,
		relationID,
	).WithContext(global.Context).Exec()

	if err != nil {
		return errors.HandleInternalError(c, "user_relations", "ScyllaDB: "+err.Error())
	}

	err = global.Session.Query(`
		UPDATE user_relations 
		SET 
		friend = ?,
		requested = ?,
		active = ? 
		WHERE user_id = ? AND relation_id = ?;`,
		false,
		false,
		false,
		relationID,
		userID,
	).WithContext(global.Context).Exec()

	if err != nil {
		return errors.HandleInternalError(c, "user_relations", "ScyllaDB: "+err.Error())
	}

	if err = messages.RelationRemove(c, relationID); err != nil {
		return err
	}

	return helpers.OKResponse(c)
}
