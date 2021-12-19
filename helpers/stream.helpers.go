package helpers

import (
	"PROJECT_H_server/errors"
	"PROJECT_H_server/global"
	"bytes"
	"fmt"

	"github.com/gocql/gocql"
	minio "github.com/minio/minio-go/v7"
)

// GetAudio gets a certain audio clip based on level
func GetAudio(userID string, chainID string, messageID string, level string, seen string, requestID string) (*bytes.Buffer, error) {

	if level == "0" && seen == "0" && userID != requestID {
		fmt.Println("LEVEL 0 AND NOT USER AND NOT SEEN")
		go func() {
			msgID, err := gocql.ParseUUID(messageID)
			if err != nil {
				errors.HandleComplexError("MessageID", "invalid")
				return
			}

			chnID, err := gocql.ParseUUID(chainID)
			if err != nil {
				errors.HandleComplexError("MessageID", "invalid")
				return
			}
			err = global.Session.Query(`
				UPDATE chains SET seen = ? WHERE chain_id = ? AND created = ?;`,
				true,
				chnID.String(),
				msgID.Time(),
			).WithContext(global.Context).Exec()

			if err != nil {
				errors.HandleComplexError("chains", "ScyllaDB: "+err.Error())
				return
			}

			err = global.Session.Query(`
				UPDATE user_relations SET last_seen = ? WHERE user_id = ? AND created = ? IF last_seen < ?;`,
				msgID.Time().UTC(),
				userID,
				chnID.Time(),
				msgID.Time().UTC(),
			).WithContext(global.Context).Exec()

			if err != nil {
				errors.HandleComplexError("user_relations", "ScyllaDB: "+err.Error())
				return
			}
		}()
	}

	object, err := global.MinIOClient.GetObject(global.Context, "audio-expire", messageID+"_l"+level, minio.GetObjectOptions{})
	if err != nil {
		return nil, errors.HandleComplexError("MinIO", "minio_get: "+err.Error())
	}

	data := new(bytes.Buffer)
	data.ReadFrom(object)

	return data, nil
}

// UpdateAction updates the action of a specific message
func UpdateAction(chainID string, messageID string, actionID string) error {

	msgID, err := gocql.ParseUUID(messageID)
	if err != nil {
		return errors.HandleComplexError("MessageID", "invalid")
	}

	chnID, err := gocql.ParseUUID(chainID)
	if err != nil {
		return errors.HandleComplexError("MessageID", "invalid")
	}

	action, err := ParseStringToInt(actionID)
	if err != nil {
		return errors.HandleComplexError("ActionID", "parsing")
	}

	if action < 0 || action > 5 {
		return errors.HandleComplexError("ActionID", "invalid")
	}

	err = global.Session.Query(`
		UPDATE chains SET action = ? WHERE chain_id = ? AND created = ?;`,
		action,
		chnID.String(),
		msgID.Time(),
	).WithContext(global.Context).Exec()

	if err != nil {
		return errors.HandleComplexError("chains", "ScyllaDB: "+err.Error())
	}

	return nil
}
