package global

import (
	"context"
	"crypto/rsa"
	"log"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/go-redis/redis/v8"
	"github.com/gocql/gocql"
	minio "github.com/minio/minio-go/v7"
)

// Logger for global logging
var Logger *log.Logger

// Session for global cassandra cql session
var Session *gocql.Session

// RedisClient for global redis queries
var RedisClient *redis.Client

// MinIOClient for global min io access
var MinIOClient *minio.Client

// PrivateKey used to decrypt assymetrical encryptions
var PrivateKey *rsa.PrivateKey

// JwtKey used to sign jwt tokens
var JwtKey *rsa.PrivateKey

// JwtParseKey used to parse jwt tokens
var JwtParseKey *rsa.PublicKey

// RefreshTokenDuration determines the lenght of a refresh token (60 days)
var RefreshTokenDuration time.Duration = time.Hour * 24 * 60

// Context is the default context
var Context = context.Background()

// Validator validates incoming bodys of data
var Validator = validator.New()

// // FFMPEGConf points to ffmpeg path
// var FFMPEGConf = &ffmpeg.Config{
// 	FfmpegBinPath:   "ffmpeg",
// 	FfprobeBinPath:  "ffprobe",
// 	ProgressEnabled: true,
// }

// // ffmpegConf := &ffmpeg.Config{
// // 	FfmpegBinPath:   "/usr/local/bin/ffmpeg",
// // 	FfprobeBinPath:  "/usr/local/bin/ffprobe",
// // 	ProgressEnabled: true,
// // }
