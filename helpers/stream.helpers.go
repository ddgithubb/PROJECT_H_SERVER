package helpers

import (
	"PROJECT_H_server/global"
	"bytes"
	Errors "errors"

	minio "github.com/minio/minio-go/v7"
)

func GetAudio(messageID string, level string, seen string) (*bytes.Buffer, error) {

	var bucket string

	if seen == "0" {
		bucket = "audio"
	} else {
		bucket = "audio_expire"
	}

	object, err := global.MinIOClient.GetObject(global.Context, bucket, messageID+"_l"+level, minio.GetObjectOptions{})
	if err != nil {
		return nil, Errors.New("minio_get: " + err.Error())
	}

	data := new(bytes.Buffer)
	data.ReadFrom(object)

	return data, nil
}
