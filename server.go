package main

import (
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"io/ioutil"
	"log"
	"os"

	"PROJECT_H_server/config"
	"PROJECT_H_server/errors"
	"PROJECT_H_server/global"
	"PROJECT_H_server/messages"
	"PROJECT_H_server/routes"
	"PROJECT_H_server/socket"

	redis "github.com/go-redis/redis/v8"
	"github.com/gocql/gocql"
	fiber "github.com/gofiber/fiber/v2"
	minio "github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/minio/minio-go/v7/pkg/lifecycle"
)

func init() {
	internalErrorsFile, err := os.OpenFile("internal_errors.txt", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0666)
	errors.HandleFatalError(err)

	monitorErrorsFile, err := os.OpenFile("monitor_logs.txt", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0666)
	errors.HandleFatalError(err)

	global.InternalLogger = log.New(internalErrorsFile, "", log.LstdFlags)
	global.MonitorLogger = log.New(monitorErrorsFile, "", log.LstdFlags)

	data, err := ioutil.ReadFile("./config.json")
	errors.HandleFatalError(err)

	err = json.Unmarshal(data, &config.Config)
	errors.HandleFatalError(err)

	privateKeyStream, err := ioutil.ReadFile("./private_key.pem")
	errors.HandleFatalError(err)
	block, _ := pem.Decode(privateKeyStream)
	global.PrivateKey, _ = x509.ParsePKCS1PrivateKey(block.Bytes)

	jwtKeyStream, err := ioutil.ReadFile("./jwt_key.pem")
	errors.HandleFatalError(err)
	block, _ = pem.Decode(jwtKeyStream)
	global.JwtKey, _ = x509.ParsePKCS1PrivateKey(block.Bytes)

	jwtKeyStream, err = ioutil.ReadFile("./jwt_key.pub")
	errors.HandleFatalError(err)
	block, _ = pem.Decode(jwtKeyStream)
	global.JwtParseKey, _ = x509.ParsePKCS1PublicKey(block.Bytes)

	global.MinIOClient, err = minio.New("127.0.0.1:9000", &minio.Options{
		Creds:  credentials.NewStaticV4(config.Config.MinIO.User, config.Config.MinIO.Password, ""),
		Secure: false, //true
	})
	errors.HandleFatalError(err)

	exists, err := global.MinIOClient.BucketExists(global.Context, "audio")
	errors.HandleFatalError(err)
	if !exists {
		global.MinIOClient.MakeBucket(global.Context, "audio", minio.MakeBucketOptions{Region: "us-east-1"})
	}

	config := lifecycle.NewConfiguration()
	config.Rules = []lifecycle.Rule{
		{
			ID:     "audio-expire",
			Status: "Enabled",
			Expiration: lifecycle.Expiration{
				Days: 1,
			},
		},
	}

	exists, err = global.MinIOClient.BucketExists(global.Context, "audio-expire")
	errors.HandleFatalError(err)
	if !exists {
		global.MinIOClient.MakeBucket(global.Context, "audio-expire", minio.MakeBucketOptions{Region: "us-east-1"})
		global.MinIOClient.SetBucketLifecycle(global.Context, "audio-expire", config)
	}

	global.RedisClient = redis.NewClient(&redis.Options{
		Addr:     "127.0.0.1:6379",
		Password: "",
		DB:       0,
	})

	cluster := gocql.NewCluster("127.0.0.1:9042")
	cluster.Keyspace = "projecthdb"
	global.Session, err = cluster.CreateSession()
	errors.HandleFatalError(err)
	fmt.Println("ScyllaDB initialized")
	fmt.Printf("Keyspace: %s\n\n", cluster.Keyspace)

	// err = global.Session.Query(`
	// 	DROP TABLE projecthdb.users;
	// `).Exec()

	// errors.HandleFatalError(err)

	// err = global.Session.Query(`
	// 	DROP TABLE projecthdb.user_relations;
	// `).Exec()

	// errors.HandleFatalError(err)

	// err = global.Session.Query(`
	// 	DROP TABLE projecthdb.chains;
	// `).Exec()

	errors.HandleFatalError(err)

	err = global.Session.Query(`
		CREATE TABLE IF NOT EXISTS projecthdb.users (
			email text,
			user_id uuid,
			password_hash text,
			created timestamp,
			username text,
			PRIMARY KEY (email)) 
		WITH compaction = { 'class' :  'LeveledCompactionStrategy'  };
	`).Exec()

	errors.HandleFatalError(err)

	err = global.Session.Query(`
		CREATE TABLE IF NOT EXISTS projecthdb.users_private (
			user_id uuid,
			username text,
			statement text,
			PRIMARY KEY (user_id)) 
		WITH compaction = { 'class' :  'LeveledCompactionStrategy'  };
	`).Exec()

	errors.HandleFatalError(err)

	err = global.Session.Query(`
		CREATE TABLE IF NOT EXISTS projecthdb.users_public (
			username text,
			user_id uuid,
			statement text,
			PRIMARY KEY (username)) 
		WITH compaction = { 'class' :  'LeveledCompactionStrategy'  };
	`).Exec()

	errors.HandleFatalError(err)

	// err = global.Session.Query(`
	// 	CREATE TABLE IF NOT EXISTS projecthdb.user_devices (
	// 		user_id uuid,
	// 		created timestamp,
	// 		device_token text,
	// 		active boolean,
	// 		PRIMARY KEY (user_id))
	// 	WITH compaction = { 'class' :  'LeveledCompactionStrategy'  };
	// `).Exec()

	// errors.HandleFatalError(err)

	// CHANGE TO CREATED
	err = global.Session.Query(`
		CREATE TABLE IF NOT EXISTS projecthdb.user_relations (
			user_id uuid,
			relation_id uuid,
			created timestamp,
			chain_id timeuuid,
			relation_username text,
			last_recv timestamp,
			last_seen timestamp,
			friend boolean,
			requested boolean,
			active boolean,
			PRIMARY KEY (user_id, relation_id)) 
		WITH compaction = { 'class' :  'LeveledCompactionStrategy'  };
	`).Exec()

	errors.HandleFatalError(err)

	err = global.Session.Query(`
		CREATE TABLE IF NOT EXISTS projecthdb.chains_users (
			chain_id uuid,
			user_id uuid,
			created timestamp,
			PRIMARY KEY (chain_id, user_id)) 
		WITH compaction = { 'class' :  'SizeTieredCompactionStrategy'  };
	`).Exec()

	err = global.Session.Query(`
		CREATE TABLE IF NOT EXISTS projecthdb.chains (
			chain_id uuid,
			created timestamp,
			message_id uuid,
			user_id uuid,
			expires timestamp,
			type int,
			seen boolean,
			display text,
			duration int,
			PRIMARY KEY (chain_id, created)) 
		WITH 
		CLUSTERING ORDER BY (created DESC) AND 
		compaction = { 'class' :  'SizeTieredCompactionStrategy'  };
	`).Exec() //BYPASS CACHE

	errors.HandleFatalError(err)

}

func main() {

	defer global.Session.Close()

	app := fiber.New()
	defer app.Shutdown()

	socket.InitializeSocketConn()
	defer socket.CloseSocketConn()

	messages.InitializeApiConn()
	defer messages.CloseApiConn()

	routes.SetRoutes(app)

	fmt.Println("Starting server on port: " + config.Config.Port)
	log.Fatal(app.Listen(config.Config.Port))

}
