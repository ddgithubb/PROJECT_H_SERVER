package services

import (
	"PROJECT_H_server/errors"
	"PROJECT_H_server/global"
	"PROJECT_H_server/helpers"
	"time"

	"github.com/gocql/gocql"
	"github.com/gofiber/fiber/v2"
)

// Request requests selected user by id
func Request(c *fiber.Ctx) error {

	userID := c.Locals("userid").(string)
	requestID := c.Locals("requestid").(string)
	reqUsername := c.Locals("reqUsername").(string)
	username := c.Query("username")

	chainID := gocql.TimeUUID()
	created := chainID.Time()

	err := global.Session.Query(`
		INSERT INTO user_relations (
			user_id,
			created,
			relation_id,
			relation_username,
			chain_id,
			last_recv,
			last_seen,
			friend,
			requested)
		VALUES(?,?,?,?,?,?,?,?,?) 
		IF NOT EXISTS;`,
		userID,
		created,
		requestID,
		reqUsername,
		chainID,
		gocql.UnsetValue,
		gocql.UnsetValue,
		gocql.UnsetValue,
		true,
	).WithContext(global.Context).Exec()

	if err != nil {
		return errors.HandleInternalError(c, "user_relations", "ScyllaDB: "+err.Error())
	}

	err = global.Session.Query(`
		INSERT INTO user_relations (
			user_id,
			created,
			relation_id,
			relation_username,
			chain_id,
			last_recv,
			last_seen,
			friend,
			requested)
		VALUES(?,?,?,?,?,?,?,?,?) 
		IF NOT EXISTS;`,
		requestID,
		created,
		userID,
		username,
		chainID,
		gocql.UnsetValue,
		gocql.UnsetValue,
		gocql.UnsetValue,
		false,
	).WithContext(global.Context).Exec()

	if err != nil {
		return errors.HandleInternalError(c, "user_relations", "ScyllaDB: "+err.Error())
	}

	return helpers.ReturnData(c, struct{ ChainID string }{ChainID: chainID.String()})
}

// RemoveRelations removes relation selected user by id
func RemoveRelation(c *fiber.Ctx) error {

	userID := c.Locals("userid").(string)
	requestID := c.Locals("requestid").(string)

	err := global.Session.Query(`
		DELETE FROM user_relations WHERE user_id = ?;`,
		userID,
	).WithContext(global.Context).Exec()

	if err != nil {
		return errors.HandleInternalError(c, "user_relations", "ScyllaDB: "+err.Error())
	}

	err = global.Session.Query(`
		DELETE FROM user_relations WHERE user_id = ?;`,
		requestID,
	).WithContext(global.Context).Exec()

	if err != nil {
		return errors.HandleInternalError(c, "user_relations", "ScyllaDB: "+err.Error())
	}

	return helpers.ReturnOKData(c)
}

// Accept accepts request from user by id
func Accept(c *fiber.Ctx) error {

	userID := c.Locals("userid").(string)
	requestID := c.Locals("requestid").(string)

	chainID, err := gocql.ParseUUID(c.Query("chainid"))
	if err != nil {
		return errors.HandleBadRequestError(c, "ChainID", "invalid")
	}
	curTime := time.Now().UTC()

	err = global.Session.Query(`
		UPDATE user_relations 
		SET 
		last_recv = ?,
		last_seen = ?,
		friend = ? 
		WHERE user_id = ? AND created = ?;`,
		curTime,
		curTime,
		true,
		userID,
		chainID.Time(),
	).WithContext(global.Context).Exec()

	if err != nil {
		return errors.HandleInternalError(c, "user_relations", "ScyllaDB: "+err.Error())
	}

	err = global.Session.Query(`
		UPDATE user_relations 
		SET 
		last_recv = ?,
		last_seen = ?,
		friend = ? 
		WHERE user_id = ? AND created = ?;`,
		curTime,
		curTime,
		true,
		requestID,
		chainID.Time(),
	).WithContext(global.Context).Exec()
	if err != nil {
		return errors.HandleInternalError(c, "user_relations", "ScyllaDB: "+err.Error())
	}

	return helpers.ReturnOKData(c)
}
