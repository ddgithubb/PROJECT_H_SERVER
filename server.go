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
	"PROJECT_H_server/routes"

	"github.com/go-redis/redis/v8"
	"github.com/gocql/gocql"
	"github.com/gofiber/fiber/v2"
	minio "github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/minio/minio-go/v7/pkg/lifecycle"
	"github.com/pebbe/zmq4"
)

func init() {
	file, err := os.OpenFile("server-logs.txt", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0666)
	errors.HandleFatalError(err)

	global.Logger = log.New(file, "", log.LstdFlags)

	data, err := ioutil.ReadFile("./config.json")
	errors.HandleFatalError(err)

	err = json.Unmarshal(data, &config.Config)
	errors.HandleFatalError(err)

	// cert, err := tls.LoadX509KeyPair("EmailCertificate.crt", "EmailKey.key")
	// if err != nil {
	// 	log.Println(err)
	// 	return
	// }

	// smtpServer := mail.NewSMTPClient()
	// smtpServer.Host = config.Config.SMTP.Host
	// smtpServer.Port = config.Config.SMTP.Port
	// smtpServer.Username = config.Config.SMTP.User
	// smtpServer.Password = config.Config.SMTP.Password
	// smtpServer.TLSConfig = &tls.Config{ServerName: config.Config.SMTP.Host, Certificates: []tls.Certificate{cert}, InsecureSkipVerify: false}
	// smtpServer.KeepAlive = true
	// smtpServer.Encryption = mail.EncryptionSTARTTLS
	// //&tls.Config{InsecureSkipVerify: true}

	// global.EmailClient, err = smtpServer.Connect()
	// errors.HandleFatalError(err)

	privateKeyStream, err := ioutil.ReadFile("./private_key.pem")
	block, _ := pem.Decode(privateKeyStream)
	global.PrivateKey, _ = x509.ParsePKCS1PrivateKey(block.Bytes)

	jwtKeyStream, err := ioutil.ReadFile("./jwt_key.pem")
	block, _ = pem.Decode(jwtKeyStream)
	global.JwtKey, _ = x509.ParsePKCS1PrivateKey(block.Bytes)

	jwtKeyStream, err = ioutil.ReadFile("./jwt_key.pub")
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

	cluster := gocql.NewCluster("127.0.0.1:8080")
	cluster.Keyspace = "projecthdb"
	global.Session, err = cluster.CreateSession()
	errors.HandleFatalError(err)
	fmt.Println("ScyllaDB initialized")
	fmt.Printf("Keyspace: %s\n\n", cluster.Keyspace)

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

	err = global.Session.Query(`
		CREATE TABLE IF NOT EXISTS projecthdb.user_devices (
			user_id uuid,
			created timestamp,
			device_token text,
			active boolean,
			PRIMARY KEY (user_id)) 
		WITH compaction = { 'class' :  'LeveledCompactionStrategy'  };
	`).Exec()

	errors.HandleFatalError(err)

	err = global.Session.Query(`
		CREATE TABLE IF NOT EXISTS projecthdb.user_relations (
			user_id uuid,
			created timestamp,
			relation_id uuid,
			relation_username text,
			chain_id uuid,
			last_recv timestamp,
			last_seen timestamp,
			friend boolean,
			requested boolean,
			PRIMARY KEY (user_id, created)) 
		WITH 
		CLUSTERING ORDER BY (created DESC) AND 
		compaction = { 'class' :  'LeveledCompactionStrategy'  };
	`).Exec()

	errors.HandleFatalError(err)

	// err = global.Session.Quer(`
	// 	DROP TABLE projecthdb.chais;
	// `).Exe()

	err = global.Session.Query(`
		CREATE TABLE IF NOT EXISTS projecthdb.chains (
			chain_id uuid,
			created timestamp,
			user_id uuid,
			message_id uuid,
			duration int,
			seen boolean,
			action int,
			display text,
			PRIMARY KEY (chain_id, created)) 
		WITH 
		CLUSTERING ORDER BY (created DESC) AND 
		compaction = { 'class' :  'SizeTieredCompactionStrategy'  };
	`).Exec() //BYPASS CACHE

	errors.HandleFatalError(err)

	// err = global.Session.Query(`
	// 	CREATE TABLE IF NOT EXISTS projecthdb.audio_clips (
	// 		audio_id uuid,
	// 		audio blob,
	// 		PRIMARY KEY (audio_id))
	// 	WITH
	// 	compaction = {
	// 		'class' :  'TimeWindowCompactionStrategy',
	// 		'compaction_window_size': '1',
	// 		'compaction_window_unit': 'DAYS'
	// 	} AND
	// 	default_time_to_live = 2592000 AND
	// 	gc_grace_seconds = 0 AND
	// 	caching = {'enabled': 'false'};
	// `).Exec() //BYPASS CACHE

	// errors.HandleFatalError(err)

}

func main() {

	defer global.Session.Close()

	go func() {
		xSub, err := zmq4.NewSocket(zmq4.XSUB)
		defer xSub.Close()
		err = xSub.Bind("tcp://*:" + config.Config.PubPort)
		errors.HandleFatalError(err)

		xPub, err := zmq4.NewSocket(zmq4.XPUB)
		defer xPub.Close()
		err = xPub.Bind("tcp://*:" + config.Config.SubPort)
		errors.HandleFatalError(err)

		log.Fatal(zmq4.Proxy(xSub, xPub, nil))
	}()

	app := fiber.New()

	routes.SetRoutes(app)

	fmt.Println("Starting server on port: " + config.Config.Port)
	log.Fatal(app.Listen(config.Config.Port))

}
