package services

import (
	"PROJECT_H_server/errors"
	"PROJECT_H_server/global"
	"PROJECT_H_server/helpers"
	"bytes"
	"fmt"
	"mime/multipart"
	"os/exec"
	"strconv"

	"github.com/gocql/gocql"
	"github.com/gofiber/fiber/v2"
	minio "github.com/minio/minio-go/v7"
)

// Authenticate authenticates users with a refresh token
func Authenticate(c *fiber.Ctx) error {

	userID := c.Locals("userid").(string)

	initUserInfo, err := helpers.GetUserInfo(c, userID)
	if initUserInfo.UserID == "" {
		return err
	}

	c.Locals("data", initUserInfo)
	return nil
}

// SendAudio sends audio message to specific friend
func SendAudio(c *fiber.Ctx) error {

	form := c.Locals("multipart").(*multipart.Form)
	// userID := c.Locals("userid").(string)
	chainID := c.Query("chainID")
	display := c.Query("display")
	duration, err := strconv.ParseInt(c.Query("duration"), 10, 64)
	if err != nil {
		return errors.HandleInternalError(c, "parse_duration", err.Error())
	}
	if duration <= 200 {
		return errors.HandleBadRequestError(c, "Duration", "too short")
	}
	messageID := gocql.TimeUUID()

	fmt.Println(chainID, display, duration)

	// err := global.Session.Query(`
	// 	INSERT INTO chains (
	// 		chain_id,
	// 		created,
	// 		user,
	// 		message_id,
	// 		seen,
	// 		action,
	// 		display)
	// 	VALUES(?,?,?,?,?,?,?);`,
	// 	chainID,
	// 	messageID.Time(),
	// 	userID,
	// 	messageID,
	// 	gocql.UnsetValue,
	// 	gocql.UnsetValue,
	// 	display,
	// ).WithContext(global.Context).Exec()

	// if err != nil {
	// 	return errors.HandleInternalError(c, "chains", "ScyllaDB: "+err.Error())
	// }

	audioFile, err := form.File["audio"][0].Open()
	if err != nil {
		return errors.HandleBadRequestError(c, "Audio", "invalid")
	}
	defer audioFile.Close()

	cmd := exec.Command("ffmpeg", "-f", "aac", "-i", "pipe:0", "-to", "00:00:05", "-c", "copy", "-f", "adts", "pipe:1")

	cmd.Stdin = audioFile

	output, err := cmd.CombinedOutput()
	if err != nil {
		return errors.HandleInternalError(c, "cmd_run", string(output))
	}

	//Cut vid 15 seconds and rest _l0, _l1, _l2, _l3 //
	_, err = global.MinIOClient.PutObject(global.Context, "audio", messageID.String()+"_l0", bytes.NewReader(output), -1, minio.PutObjectOptions{ContentType: "audio/mpeg"})
	if err != nil {
		return errors.HandleInternalError(c, "minio_put", err.Error())
	}

	if duration > 5000 {
		_, err := audioFile.Seek(0, 0)
		if err != nil {
			return errors.HandleInternalError(c, "audio_seek", err.Error())
		}

		cmd := exec.Command("ffmpeg", "-f", "aac", "-ss", "00:00:05", "-i", "pipe:0", "-c", "copy", "-f", "adts", "pipe:1")

		cmd.Stdin = audioFile

		output, err := cmd.CombinedOutput()
		if err != nil {
			return errors.HandleInternalError(c, "cmd_run", string(output))
		}

		//Cut vid 15 seconds and rest _l0, _l1, _l2, _l3 //
		_, err = global.MinIOClient.PutObject(global.Context, "audio", messageID.String()+"_l1", bytes.NewReader(output), -1, minio.PutObjectOptions{ContentType: "audio/mpeg"})
		if err != nil {
			return errors.HandleInternalError(c, "minio_put", err.Error())
		}
	}

	c.Locals("data", struct{ MessageID string }{MessageID: messageID.String()})
	return nil
}
