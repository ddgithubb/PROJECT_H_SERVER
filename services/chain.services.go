package services

import (
	"PROJECT_H_server/errors"
	"PROJECT_H_server/global"
	"PROJECT_H_server/helpers"
	"PROJECT_H_server/schemas"
	"fmt"
	"strconv"
	"time"

	"github.com/gocql/gocql"
	"github.com/gofiber/fiber/v2"
	minio "github.com/minio/minio-go/v7"
)

// GetChain gets a segment of chain for specified relation
func GetChain(c *fiber.Ctx) error {

	chainID := c.Locals("chainid").(gocql.UUID)

	request, err := strconv.ParseInt(c.Query("requestTime"), 10, 64)
	if err != nil {
		return errors.HandleInternalError(c, "parse_request_time", err.Error())
	}
	requestTime := time.UnixMilli(request)
	asc := c.Query("asc")
	desc := c.Query("desc")
	limit, err := strconv.ParseInt(c.Query("limit", "50"), 10, 64)
	if err != nil {
		return errors.HandleInternalError(c, "parse_limit", err.Error())
	}

	if asc != "true" && desc != "true" {
		newChain, err := helpers.GetChain(chainID, requestTime, false, true, limit)
		if err != nil {
			return errors.HandleInternalError(c, "helpers_get_chain", err.Error())
		}

		return c.JSON(newChain)
	} else {

		ascChain := []schemas.MessageSchema{}
		descChain := []schemas.MessageSchema{}

		if asc == "true" {
			ascChain, err = helpers.GetChain(chainID, requestTime, true, false, limit)
			if err != nil {
				return errors.HandleInternalError(c, "helpers_get_chain", err.Error())
			}
		}

		if desc == "true" {
			descChain, err = helpers.GetChain(chainID, requestTime, false, false, limit)
			if err != nil {
				return errors.HandleInternalError(c, "helpers_get_chain", err.Error())
			}
		}

		return c.JSON(append(descChain, ascChain...))
	}
}

// AddAudioMessage adds audio message to specified chain
func AddAudioMessage(c *fiber.Ctx) error {

	userID := c.Locals("userid").(string)
	chainID := c.Locals("chainid").(gocql.UUID)
	requestID := c.Locals("requestid").(string)

	form, err := c.MultipartForm()
	if err != nil {
		return errors.HandleBadRequestError(c, "Multipart", "invalid")
	}

	display := form.Value["display"][0]
	duration, err := strconv.ParseInt(form.Value["duration"][0], 10, 64)
	if err != nil {
		return errors.HandleInternalError(c, "parse_duration", err.Error())
	}

	if duration > (5*60000 + 30) {
		return errors.HandleBadRequestError(c, "Duration", "too long")
	}

	if form.File["audio"][0].Size > 1900000 {
		return errors.HandleBadRequestError(c, "Audio", "exceeding length")
	}

	audioFile, err := form.File["audio"][0].Open()
	if err != nil {
		return errors.HandleBadRequestError(c, "Audio", "invalid")
	}
	defer audioFile.Close()

	messageID := gocql.TimeUUID()
	fmt.Println("AUDIO FILE SPECS: ", chainID, display, duration, form.File["audio"][0].Size)

	_, err = global.MinIOClient.PutObject(global.Context, "audio-expire", chainID.String()+"_"+messageID.String(), audioFile, -1, minio.PutObjectOptions{ContentType: "audio/mpeg"})
	if err != nil {
		return errors.HandleInternalError(c, "minio_put", err.Error())
	}

	err = global.Session.Query(`
		INSERT INTO chains (
			chain_id,
			created,
			user_id,
			message_id,
			duration,
			seen,
			action,
			display)
		VALUES(?,?,?,?,?,?,?,?);`,
		chainID,
		messageID.Time().UTC(),
		userID,
		messageID,
		duration,
		gocql.UnsetValue,
		gocql.UnsetValue,
		display,
	).WithContext(global.Context).Exec()

	if err != nil {
		return errors.HandleInternalError(c, "chains", "ScyllaDB: "+err.Error())
	}

	err = global.Session.Query(`
		UPDATE user_relations SET last_recv = ? WHERE user_id = ? AND created = ?;`,
		messageID.Time().UTC(),
		requestID,
		chainID.Time(),
	).WithContext(global.Context).Exec()

	if err != nil {
		return errors.HandleInternalError(c, "user_relations", "ScyllaDB: "+err.Error())
	}

	err = global.Session.Query(`
		UPDATE user_relations SET last_seen = ? WHERE user_id = ? AND created = ?;`,
		messageID.Time().UTC(),
		userID,
		chainID.Time(),
	).WithContext(global.Context).Exec()

	if err != nil {
		return errors.HandleInternalError(c, "user_relations", "ScyllaDB: "+err.Error())
	}

	return c.JSON(struct {
		MessageID string
		LastSeen  int64
	}{
		MessageID: messageID.String(),
		LastSeen:  messageID.Time().UnixMilli(),
	})
}

// GetAudioMessage gets a certain audio clip based on level
func GetAudioMessage(c *fiber.Ctx) error {

	userID := c.Locals("userid").(string)
	chainID := c.Locals("chainid").(gocql.UUID)
	seen := c.Query("seen")
	messageID, err := helpers.ParseMessageUUID(c)
	if err != nil {
		return err
	}

	object, err := global.MinIOClient.GetObject(global.Context, "audio-expire", chainID.String()+"_"+messageID.String(), minio.GetObjectOptions{})
	if err != nil {
		//ExpireAt if err is not a network error
		return errors.HandleInvalidRequestError(c, "Message", "expired")
		// return errors.HandleInternalError(c, "MinIO", "minio_get: "+err.Error())
	}

	if seen == "false" {
		fmt.Println("LEVEL 0 AND NOT USER AND NOT SEEN")
		go func() {
			err = global.Session.Query(`
				UPDATE chains SET seen = ? WHERE chain_id = ? AND created = ?;`,
				true,
				chainID.String(),
				messageID.Time(),
			).WithContext(global.Context).Exec()

			if err != nil {
				errors.HandleComplexError("chains", "ScyllaDB: "+err.Error())
				return
			}

			err = global.Session.Query(`
				UPDATE user_relations SET last_seen = ? WHERE user_id = ? AND created = ? IF last_seen < ?;`,
				messageID.Time().UTC(),
				userID,
				chainID.Time(),
				messageID.Time().UTC(),
			).WithContext(global.Context).Exec()

			if err != nil {
				errors.HandleComplexError("user_relations", "ScyllaDB: "+err.Error())
				return
			}
		}()
	}

	return c.SendStream(object)
}
