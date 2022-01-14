package services

import (
	"PROJECT_H_server/errors"
	"PROJECT_H_server/global"
	"PROJECT_H_server/helpers"
	"PROJECT_H_server/schemas"
	"bytes"
	"fmt"
	"mime/multipart"
	"os/exec"
	"strconv"
	"time"

	"github.com/gocql/gocql"
	"github.com/gofiber/fiber/v2"
	minio "github.com/minio/minio-go/v7"
)

// SendAudio sends audio message to specific friend
func SendAudio(c *fiber.Ctx) error {

	form := c.Locals("multipart").(*multipart.Form)
	userID := c.Locals("userid").(string)
	requestID := c.Query("requestid")
	chainID, err := gocql.ParseUUID(c.Query("chainID"))
	if err != nil {
		return errors.HandleBadRequestError(c, "ChainID", "invalid")
	}
	display := form.Value["display"][0]
	duration, err := strconv.ParseInt(c.Query("duration"), 10, 64)
	if err != nil {
		return errors.HandleInternalError(c, "parse_duration", err.Error())
	}
	if duration <= 200 {
		return errors.HandleBadRequestError(c, "Duration", "too short")
	}
	if duration > 5.1*60000 {
		return errors.HandleBadRequestError(c, "Duration", "too long")
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
	fmt.Println(chainID, display, duration)

	cmd := exec.Command("ffmpeg", "-f", "aac", "-i", "pipe:0", "-to", "00:00:20", "-c", "copy", "-f", "adts", "pipe:1")

	cmd.Stdin = audioFile

	output, err := cmd.CombinedOutput()
	if err != nil {
		return errors.HandleInternalError(c, "cmd_run", string(output))
	}

	//Cut vid 15 seconds and rest _l0, _l1, _l2, _l3 //
	_, err = global.MinIOClient.PutObject(global.Context, "audio-expire", messageID.String()+"_l0", bytes.NewReader(output), -1, minio.PutObjectOptions{ContentType: "audio/mpeg"})
	if err != nil {
		return errors.HandleInternalError(c, "minio_put", err.Error())
	}

	if duration > 20000 {
		_, err := audioFile.Seek(0, 0)
		if err != nil {
			return errors.HandleInternalError(c, "audio_seek", err.Error())
		}

		cmd := exec.Command("ffmpeg", "-f", "aac", "-i", "pipe:0", "-ss", "00:00:20", "-to", "00:02:30", "-c", "copy", "-f", "adts", "pipe:1")

		cmd.Stdin = audioFile

		output, err := cmd.CombinedOutput()
		if err != nil {
			return errors.HandleInternalError(c, "cmd_run", string(output))
		}

		//Cut vid 15 seconds and rest _l0, _l1, _l2, _l3 //
		_, err = global.MinIOClient.PutObject(global.Context, "audio-expire", messageID.String()+"_l1", bytes.NewReader(output), -1, minio.PutObjectOptions{ContentType: "audio/mpeg"})
		if err != nil {
			return errors.HandleInternalError(c, "minio_put", err.Error())
		}
	}

	if duration > 2.5*60000 {
		_, err := audioFile.Seek(0, 0)
		if err != nil {
			return errors.HandleInternalError(c, "audio_seek", err.Error())
		}

		cmd := exec.Command("ffmpeg", "-f", "aac", "-i", "pipe:0", "-ss", "00:02:30", "-c", "copy", "-f", "adts", "pipe:1")

		cmd.Stdin = audioFile

		output, err := cmd.CombinedOutput()
		if err != nil {
			return errors.HandleInternalError(c, "cmd_run", string(output))
		}

		//Cut vid 15 seconds and rest _l0, _l1, _l2, _l3 //
		_, err = global.MinIOClient.PutObject(global.Context, "audio-expire", messageID.String()+"_l2", bytes.NewReader(output), -1, minio.PutObjectOptions{ContentType: "audio/mpeg"})
		if err != nil {
			return errors.HandleInternalError(c, "minio_put", err.Error())
		}
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

	return helpers.ReturnData(c, struct {
		MessageID string
		LastSeen  int64
	}{
		MessageID: messageID.String(),
		LastSeen:  messageID.Time().UnixMilli(),
	})
}

// GetChain gets a segment of chain for specified relation
func GetChain(c *fiber.Ctx) error {

	chainID, err := gocql.ParseUUID(c.Query("chainID"))
	if err != nil {
		return errors.HandleBadRequestError(c, "ChainID", "invalid")
	}
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

		return helpers.ReturnData(c, newChain)
	} else {

		ascChain := []schemas.ChainSchema{}
		descChain := []schemas.ChainSchema{}

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
		return helpers.ReturnData(c, append(descChain, ascChain...))
	}
}
