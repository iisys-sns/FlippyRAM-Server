package template

import (
	"log"
	"fmt"
	"strings"
	"regexp"
	"html"
	"crypto/sha256"
	"flippyram-server/src/config"
)

var c *config.HammerserverConfig

func Init(conf *config.HammerserverConfig) {
	c = conf
}

func HandleTemplate(path string, token string) *config.FileInfo {
	fileInfo := c.AccessCachedFile(path)
	if fileInfo == nil {
		log.Printf("Unable to open template file %s.", path)
		return nil
	}
	re := regexp.MustCompile(`^[0-9a-f]{64}$`)
	if !re.MatchString(token) {
		log.Printf("Submitted token does not look like a sha256 string. Ignoring...")
		return nil
	}

	modified := string(fileInfo.Content)
	modified = strings.ReplaceAll(modified, "##HASH##", html.EscapeString(token))

	var info config.FileInfo
	info.Content = []byte(modified)
	info.Hash = fmt.Sprintf("%x", sha256.Sum256(fileInfo.Content))
	info.StoresContent = true
	info.Path = ""
	return &info
}
