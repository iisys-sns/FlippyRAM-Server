package helper

import (
	"os"
	"time"
	"database/sql"
	"regexp"
	"bytes"
	"log"
	"fmt"
	"io"
	"io/ioutil"
	"errors"
	"net/http"
	"archive/zip"
	"strings"
	"crypto/sha256"
	"flippyram-server/src/config"

	_ "github.com/go-sql-driver/mysql"
)

var Datasets map[string]bool
var c *config.HammerserverConfig
var db *sql.DB

func Init(conf *config.HammerserverConfig) {
	c = conf
	Datasets = make(map[string]bool)

	files, err := ioutil.ReadDir(c.GetDatasetDirectory())
	if err != nil {
		log.Fatalf("Unable to open dataset directory %s. Error: %s", c.GetDatasetDirectory(), err.Error())
	}

	for _, file := range files {
		Datasets[strings.ReplaceAll(file.Name(), ".zip", "")] = true
	}

	dbConnectionString := ""
	dbConnectionString += c.Database.Username + ":" + c.Database.Password
	dbConnectionString += "@" + c.Database.ConnectionType + "(" + c.Database.ConnectionLocation + ")"
	dbConnectionString += "/" + c.Database.Name

	db, err = sql.Open("mysql", dbConnectionString)
	if err != nil {
		log.Fatalf("Unable to create database connection with '%s'. Error: %s", dbConnectionString, err.Error())
	}

	err = db.Ping()
	if err != nil {
		log.Fatalf("Unable to connect to database. Error: %s", err.Error())
	}
}

func VerifyZip(content []byte) bool {
	reader := bytes.NewReader(content)
	zipReader, err := zip.NewReader(reader, int64(len(content)))
	if err != nil {
		log.Printf("Unable to register zip reader for uploaded file. Error: %s", err.Error())
		return false
	}

	permittedUploadFiles := c.GetPermittedUploadFiles()
	permittedUploadWildcards := c.GetPermittedUploadWildcards()

	for _, file := range zipReader.File {
		contentType, ok := (*permittedUploadFiles)[file.FileHeader.Name]
		if !ok {
			foundMatchingWildcard := false
			for regex, cType := range *permittedUploadWildcards {
				if regex.MatchString(file.FileHeader.Name) {
					foundMatchingWildcard = true
					contentType = cType
					break
				}
			}

			if !foundMatchingWildcard {
				log.Printf("File %s in uploaded ZIP is not whitelisted. Blocking upload.", file.FileHeader.Name)
				return false
			}
		}

		if file.FileInfo().IsDir() {
			if contentType == "directory" {
				continue
			}
			log.Printf("Directory %s in uploaded ZIP does not have 'directory' content type, but '%s'", file.FileHeader.Name, contentType)
			return false
		}

		if file.FileHeader.UncompressedSize > c.GetMaxUncompressedFileSize() {
			log.Printf("File %s in uploaded ZIP is too big (%d > %d)", file.FileHeader.Name, file.FileHeader.UncompressedSize, c.GetMaxUncompressedFileSize())
			return false
		}

		fileHandler, err := file.Open()
		if err != nil {
			log.Printf("Unable to open file %s within uploaded ZIP. Error: %s", file.FileHeader.Name, err.Error())
			return false
		}

		data := make([]byte, 512)
		size, err := io.ReadFull(fileHandler, data)
		if err != nil {
			if !errors.Is(err, io.ErrUnexpectedEOF) && !errors.Is(err, io.EOF){
				log.Printf("Unable to read from file %s within uploaded ZIP. Error: %s", file.FileHeader.Name, err.Error())
				return false
			}
		}

		detectedContentType := http.DetectContentType(data[:size])
		if detectedContentType != contentType && !strings.HasPrefix(detectedContentType, contentType + ";") {
			log.Printf("File %s in ZIP has invalid content type %s (should be %s)", file.FileHeader.Name, detectedContentType, contentType)
			return false
		}
	}

	Datasets[fmt.Sprintf("%x", sha256.Sum256(content))] = true

	return true
}

func VerifyToken(token string) bool {
	re := regexp.MustCompile(`^[0-9a-f]{64}`)
	val := re.MatchString(token)
	if !val {
		log.Printf("Submitted token does not look like a sha256 string. Ignoring...")
	}

	return val
}

func RemoveFile(hash string) {
	_, ok := Datasets[hash]
	if ok {
		delete(Datasets, hash)
	}
}

func GetRandomToken() string {
	file, err := os.Open("/dev/random")
	if err != nil {
		log.Printf("Unable to open /dev/random. Error: %s", err.Error())
		return ""
	}

	buf := make([]byte, 64)
	n, err := file.Read(buf)
	if err != nil {
		log.Printf("Unable to read from /dev/random. Error: %s", err.Error())
		return ""
	}

	if n != len(buf) {
		log.Printf("Unable to read %d bytes from /dev/random. Got only %d.", len(buf), n)
		return ""
	}

	time := time.Now().String()

	randomContent := append(buf, []byte(time)...)
	token := fmt.Sprintf("%x", sha256.Sum256(randomContent))
	return token
}

func StoreToken(token string) bool {
	if !VerifyToken(token) {
		return false
	}

	rows, err := db.Query("SELECT * FROM token WHERE value = ?;", token)
	if err != nil {
		log.Printf("Unable to get token %s from database (to see if it already exists). Error: %s", token, err.Error())
		return false
	}

	cnt := 0
	for rows.Next() {
		cnt++
	}
	if cnt != 0 {
		log.Printf("The token %s does already exist in the database. Not adding it again.", token)
		return false
	}

	rows.Close()

	_, err = db.Exec("INSERT INTO token(value) VALUES (?);", token)
	if err != nil {
		log.Printf("Unable to insert token %s into database. Error: %s", token, err.Error())
		return false
	}

	return true
}
