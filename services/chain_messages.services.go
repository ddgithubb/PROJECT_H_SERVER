package services

import (
	"PROJECT_H_server/errors"
	"PROJECT_H_server/global"
	"PROJECT_H_server/helpers"
	"PROJECT_H_server/messages"
	"fmt"
	"strconv"
	"time"

	"github.com/gocql/gocql"
	"github.com/gofiber/fiber/v2"
	minio "github.com/minio/minio-go/v7"
)

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

	expireDays, err := strconv.Atoi(form.Value["expireDays"][0])
	if err != nil {
		return errors.HandleInternalError(c, "parse_expires", err.Error())
	}

	expires := time.Now().AddDate(0, 0, 60)

	if expireDays > 0 && expireDays < 60 {
		expires = time.Now().AddDate(0, 0, int(expireDays))
	}

	if form.File["audio"][0].Size > 2000000 {
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
			message_id,
			user_id,
			expires,
			type,
			seen,
			display,
			duration) 
		VALUES(?,?,?,?,?,?,?,?,?);`,
		chainID,
		messageID.Time().UTC(),
		messageID,
		userID,
		expires,
		0,
		gocql.UnsetValue,
		display,
		duration,
	).WithContext(global.Context).Exec()

	if err != nil {
		return errors.HandleInternalError(c, "chains", "ScyllaDB: "+err.Error())
	}

	err = global.Session.Query(`
		UPDATE user_relations SET last_recv = ? WHERE user_id = ? AND relation_id = ?;`,
		messageID.Time().UTC(),
		requestID,
		userID,
	).WithContext(global.Context).Exec()

	if err != nil {
		return errors.HandleInternalError(c, "user_relations", "ScyllaDB: "+err.Error())
	}

	err = global.Session.Query(`
		UPDATE user_relations SET last_seen = ? WHERE user_id = ? AND relation_id = ?;`,
		messageID.Time().UTC(),
		userID,
		requestID,
	).WithContext(global.Context).Exec()

	if err != nil {
		return errors.HandleInternalError(c, "user_relations", "ScyllaDB: "+err.Error())
	}

	if err = messages.SendMessage(c, requestID, chainID.String(), messageID.String(), messageID.Time().UnixMilli(), expires.UnixMilli(), 0, display, duration); err != nil {
		return err
	}

	return c.JSON(struct {
		MessageID string
		LastSeen  int64
	}{
		MessageID: messageID.String(),
		LastSeen:  messageID.Time().UnixMilli(),
	})
}

// GetAudioMessage gets a certain audio clip
func GetAudioMessage(c *fiber.Ctx) error {

	userID := c.Locals("userid").(string)
	chainID := c.Locals("chainid").(gocql.UUID)
	seen := c.Query("seen")
	messageID, err := helpers.ParseParamMessageUUID(c)
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
				UPDATE user_relations SET last_seen = ? WHERE user_id = ? AND chain_id = ? IF last_seen < ?;`,
				messageID.Time().UTC(),
				userID,
				chainID,
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
