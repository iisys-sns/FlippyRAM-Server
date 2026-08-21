package config

import (
	"crypto/sha256"
	"encoding/json"
	"os"
	"log"
	"fmt"
	"strings"
	"time"
	"regexp"
)

type HammerserverConfig struct {
	ListenAddress							string				`json:"listen_address"`
	HTTPListenAddress					string				`json:"http_redirect_listen_address"`
	WebRoot										string				`json:"web_root"`
	WhitelistPath							string				`json:"whitelist_path"`
	PermittedUploadsPath			string				`json:"permitted_uploads_path"`
	IndexPath									string				`json:"index"`
	TokenTemplatePath					string				`json:"token_template"`
	DatasetDirectory					string				`json:"dataset_directory"`
	CertificatePath						string				`json:"certificate_path"`
	KeyPath										string				`json:"key_path"`
	MaxUploadSize							int						`json:"max_upload_size"`
	MaxUncompressedFileSize		uint32				`json:"max_uncompressed_file_size"`
	ReadTimeout								time.Duration `json:"read_timeout"`
	MaxHeaderBytes						int						`json:"max_header_bytes"`
	StaticResponseHeaders			[]string			`json:"static_response_headers"`
	MaxCacheFilesize					int						`json:"max_cache_filesize"`
	PermittedUploadFiles			map[string]string
	PermittedUploadWildcards	map[*regexp.Regexp]string
	Database									struct {
		Username								string        `json:"username"`
		Password								string        `json:"password"`
		ConnectionType					string        `json:"connection_type"`
		ConnectionLocation			string        `json:"connection_location"`
		Name										string        `json:"name"`
	}                                       `json:"database"`
}

type FileInfo struct {
	StoresContent bool
	Path string
	Content []byte
	Hash string
}

var Files map[string]*FileInfo
var ZipFiles map[string]*FileInfo

func init() {
	Files = make(map[string]*FileInfo)
	ZipFiles = make(map[string]*FileInfo)
}

func (c *HammerserverConfig)AccessCachedFile(path string) *FileInfo {
	fileInfo, ok := Files[path]
	if ok {
		return fileInfo
	}

	bytes, err := os.ReadFile(c.WebRoot + path)
	if err != nil {
		log.Printf("Unable to read content of file '%s', ignoring. Error: %s", path, err.Error())
		return nil
	}

	var newFileInfo FileInfo
	newFileInfo.Path = c.WebRoot + path
	newFileInfo.Hash = fmt.Sprintf("%x", sha256.Sum256(bytes))
	if len(bytes) <= c.MaxCacheFilesize {
		newFileInfo.Content = bytes
		newFileInfo.StoresContent = true
	} else {
		newFileInfo.Content = []byte{}
		newFileInfo.StoresContent = false
	}

	Files[path] = &newFileInfo

	return &newFileInfo
}

func (c *HammerserverConfig)AccessCachedZipFile(hash string) *FileInfo {
	fileInfo, ok := ZipFiles[hash]
	if ok {
		return fileInfo
	}

	bytes, err := os.ReadFile(c.DatasetDirectory + hash + ".zip")
	if err != nil {
		log.Printf("Unable to read content of ZIP file '%s', ignoring. Error: %s", c.DatasetDirectory + hash + ".zip", err.Error())
		return nil
	}

	var newFileInfo FileInfo
	newFileInfo.Path = c.DatasetDirectory + hash + ".zip"
	newFileInfo.Hash = fmt.Sprintf("%x", sha256.Sum256(bytes))
	if len(bytes) <= c.MaxCacheFilesize {
		newFileInfo.Content = bytes
		newFileInfo.StoresContent = true
	} else {
		newFileInfo.Content = []byte{}
		newFileInfo.StoresContent = false
	}

	ZipFiles[hash] = &newFileInfo

	return &newFileInfo
}

func ParseConfig(configpath string) *HammerserverConfig {
	var c HammerserverConfig

	data, err := os.ReadFile(configpath)
	if err != nil {
		log.Fatalf("Unable to open config file '%s'. Error: %s", configpath, err.Error())
	}

	err = json.Unmarshal(data, &c)
	if err != nil {
		log.Fatalf("Unable to parse config file '%s'. Error: %s", configpath, err.Error())
	}

	c.loadPermittedUploadFiles()

	return &c
}

func(c *HammerserverConfig) GetWhitelist() [][]string {
	ret := [][]string{}

	data, err := os.ReadFile(c.WhitelistPath)
	if err != nil {
		log.Fatalf("Unable to open whitelist file '%s'. Error: %s", c.WhitelistPath, err.Error())
	}

	for _, line := range strings.Split(string(data[:len(data)-1]), "\n") {
		parts := strings.Split(line, "|")
			if len(parts) != 2 {
				log.Fatalf("Unable to parse whitelist line '%s'. There should be a path and content type delimited by '|'.", line)
			}
		ret = append(ret, parts)
	}
	return ret
}

func(c *HammerserverConfig) loadPermittedUploadFiles() {
	c.PermittedUploadFiles = make(map[string]string)
	c.PermittedUploadWildcards = make(map[*regexp.Regexp]string)
	
	data, err := os.ReadFile(c.PermittedUploadsPath)
	if err != nil {
		log.Fatalf("Unable to open upload path file '%s'. Error: %s", c.WhitelistPath, err.Error())
	}

	if len(data) == 0 {
		return
	}

	for _, line := range strings.Split(string(data[:len(data)-1]), "\n") {
		parts := strings.Split(line, "|")
			if len(parts) != 2 {
				log.Fatalf("Unable to parse upload path line '%s'. There should be a path and content type delimited by '|'.", line)
			}
		if strings.Contains(parts[0], "*") {
			re := regexp.MustCompile("^" + strings.ReplaceAll(parts[0], "*", ".*") + "$")
			c.PermittedUploadWildcards[re] = parts[1]
		} else {
			c.PermittedUploadFiles[parts[0]] = parts[1]
		}
	}
}

func(c *HammerserverConfig) GetPermittedUploadFiles() *map[string]string {
	return &c.PermittedUploadFiles
}

func(c *HammerserverConfig) GetPermittedUploadWildcards() *map[*regexp.Regexp]string {
	return &c.PermittedUploadWildcards
}

func(c *HammerserverConfig) GetListenAddress() string {
	return c.ListenAddress
}

func(c *HammerserverConfig) GetHTTPListenAddress() string {
	return c.HTTPListenAddress
}

func(c *HammerserverConfig) GetWebRoot() string {
	return c.WebRoot
}

func(c *HammerserverConfig) GetIndexPath() string {
	return c.IndexPath
}

func(c *HammerserverConfig) GetTokenTemplatePath() string {
	return c.TokenTemplatePath
}

func(c *HammerserverConfig) GetDatasetDirectory() string {
	return c.DatasetDirectory
}

func(c *HammerserverConfig) GetCertificatePath() string {
	return c.CertificatePath
}

func (c *HammerserverConfig) GetKeyPath() string {
	return c.KeyPath
}

func (c *HammerserverConfig) GetMaxUploadSize() int {
	return c.MaxUploadSize
}

func (c *HammerserverConfig) GetMaxUncompressedFileSize() uint32 {
	return c.MaxUncompressedFileSize
}

func (c *HammerserverConfig) GetReadTimeout() time.Duration {
	return c.ReadTimeout * time.Second
}

func (c *HammerserverConfig) GetMaxHeaderBytes() int {
	return c.MaxHeaderBytes
}

func (c *HammerserverConfig) GetStaticResponseHeaders() []string {
	return c.StaticResponseHeaders
}

func (c *HammerserverConfig) RemoveFile(hash string) bool{
	_, ok := ZipFiles[hash]
	if ok {
		delete(ZipFiles, hash)
	}

	err := os.Remove(c.DatasetDirectory + hash + ".zip")
	if err != nil {
		log.Printf("Unable to remove file %s. Error: %s", c.DatasetDirectory + hash + ".zip", err.Error())
		return false
	}

	return true
}
