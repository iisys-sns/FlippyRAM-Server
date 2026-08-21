package main

import (
	"runtime"
	"io/ioutil"
	"time"
	"os"
	"fmt"
	"log"
	"errors"
	"strings"
	"net/http"
	"crypto/sha256"
	"crypto/tls"
	"flippyram-server/src/config"
	"flippyram-server/src/helper"
	"flippyram-server/src/template"
)

func setHeaders(contentType string, fileInfo *config.FileInfo, c *config.HammerserverConfig, w http.ResponseWriter) {
}

func sendResponse(contentType string, fileInfo *config.FileInfo, c *config.HammerserverConfig, w http.ResponseWriter, r *http.Request) {
	if fileInfo == nil {
		log.Printf("Unable to send 'nil' file info. Skipping...")
		http.Redirect(w, r, "/error.html", http.StatusSeeOther)
		return
	}
	w.Header().Set("Content-Type", contentType)
	w.Header().Set("Last-Modified", time.Now().Format("Mon, 02 Jan 2006 15:04:05 MST"))
	w.Header().Set("Expires", time.Now().Add(time.Hour * 1).Format("Mon, 02 Jan 2006 15:04:05 MST"))
	w.Header().Set("Date", time.Now().Add(1 * time.Second).Format("Mon, 02 Jan 2006 15:04:05 MST"))
	w.Header().Set("Cache-Control", "public, max-age=3600")
	w.Header().Set("ETag", fileInfo.Hash)
	w.Header().Set("If-None-Match", fileInfo.Hash)

	for _, header := range c.GetStaticResponseHeaders() {
		w.Header().Set(strings.Split(header, ": ")[0], strings.Split(header, ": ")[1])
	}

	if fileInfo.StoresContent {
		w.Write(fileInfo.Content)
	} else {
		fileStat, err := os.Stat(fileInfo.Path)
		if err != nil {
			log.Printf("Unable to stat file %s. Error: %s", fileInfo.Path, err.Error())
			return
		}
		if fileStat.Mode().IsRegular() {
			http.ServeFile(w, r, fileInfo.Path)
		} else {
			log.Printf("File %s is not a regular file. Skipping...", fileInfo.Path)
		}
	}
}

func main() {
	runtime.GOMAXPROCS(runtime.NumCPU())
	if len(os.Args) != 2 {
		log.Fatalf("Usage: %s path/to/config/file", os.Args[0])
	}

	c := config.ParseConfig(os.Args[1])
	helper.Init(c)
	template.Init(c)
	
	initialFile := ""
	initialContentType := ""
	for _, line := range c.GetWhitelist() {
		file := line[0]
		contentType := line[1]
		if len(initialFile) == 0 {
			initialFile = file
			initialContentType = contentType
		}


		// Deliver whitelisted files
		http.HandleFunc("/" + file, func(w http.ResponseWriter, r *http.Request) {
			fileInfo := c.AccessCachedFile(file)
			if fileInfo == nil {
				log.Printf("Unable to access file %s. Skipping...", file)
				return
			}
			sendResponse(contentType, fileInfo, c, w, r)
		})
	}

	// Token pages
	http.HandleFunc("/token/", func(w http.ResponseWriter, r *http.Request) {
		token := strings.TrimPrefix(r.URL.Path, "/token/")
		var fileInfo *config.FileInfo

		if token == "" {
			http.Redirect(w, r, "/", http.StatusSeeOther)
			return
		}

		if strings.HasSuffix(token, "/") {
			token = token[:len(token)-1]
		}

		if !helper.VerifyToken(token) {
			log.Printf("Token %s is invalid. Skipping...", c.IndexPath)
			http.Redirect(w, r, "/error.html", http.StatusSeeOther)
			return
		}

		fileInfo = template.HandleTemplate(c.GetTokenTemplatePath(), token)

		sendResponse("text/html", fileInfo, c, w, r)
	})

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fileInfo := c.AccessCachedFile(initialFile)
		if(fileInfo == nil) {
			log.Printf("Unable to access file %s. Skipping...", initialFile)
			return
		}
		sendResponse(initialContentType, fileInfo, c, w, r)
	})

	http.HandleFunc("/upload/", func(w http.ResponseWriter, r *http.Request) {
		r.ParseMultipartForm(int64(c.GetMaxUploadSize()))
		file, handler, err := r.FormFile("file")
		if err != nil {
			log.Printf("Unable to receive uploaded multipart form. Error: %s", err.Error())
			http.Redirect(w, r, "/error.html", http.StatusSeeOther)
			return
		}

		data, err := ioutil.ReadAll(file)
		if err != nil {
			log.Printf("Unable to read uploaded file. Error: %s", err.Error())
			http.Redirect(w, r, "/error.html", http.StatusSeeOther)
			return
		}

		size := handler.Size
		if size > 512 {
			size = 512
		}
		detectedContentType := http.DetectContentType(data[:size])
		if detectedContentType != "application/zip" {
			log.Printf("MIME type of uploaded file does not match 'application/zip'. It is: '%s'", detectedContentType)
			http.Redirect(w, r, "/error.html", http.StatusSeeOther)
			return
		}

		if !helper.VerifyZip(data) {
			log.Printf("Uploaded file does not follow the assumed directory structure.")
			http.Redirect(w, r, "/error.html", http.StatusSeeOther)
			return
		}

		fileHash := fmt.Sprintf("%x", sha256.Sum256(data))
		filePath := c.GetDatasetDirectory() + fileHash + ".zip"
		_, err = os.Stat(filePath)
		if !errors.Is(err, os.ErrNotExist) {
			if err == nil {
				log.Printf("The file %s does already exist. Not uploading again.", filePath)
				http.Redirect(w, r, "/error.html", http.StatusSeeOther)
			} else {
			// TODO: Add info that upload filed (e.g. error page)
				log.Printf("Error getting FS stat for file %s: %s", filePath, err.Error())
				http.Redirect(w, r, "/error.html", http.StatusSeeOther)
			}
			return
		}

		outFile, err := os.OpenFile(filePath, os.O_CREATE|os.O_WRONLY|os.O_EXCL, 0640)
		if err != nil {
			// TODO: Add info that upload filed (e.g. error page)
			log.Printf("Unable to create file %s. Error: %s", filePath, err.Error())
			http.Redirect(w, r, "/error.html", http.StatusSeeOther)
			return
		}

		defer outFile.Close()
		outFile.Write(data)

		token := helper.GetRandomToken()
		helper.StoreToken(token)
		http.Redirect(w, r, "/token/" + token, http.StatusSeeOther)
	})

	// HTTP server for redirects
	go func() {
		httpSrv := &http.Server {
			Addr: c.GetHTTPListenAddress(),
			Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				host := strings.Split(r.Host, ":")[0] + ":" + strings.Split(c.GetListenAddress(), ":")[1]
				http.Redirect(w, r, "https://" + host + r.URL.String(), http.StatusMovedPermanently)
			}),
			ReadTimeout: c.GetReadTimeout(),
			MaxHeaderBytes: c.GetMaxHeaderBytes(),
		}
		log.Fatal(httpSrv.ListenAndServe())
	}()

	// HTTPS server
	srv := &http.Server {
		Addr: c.GetListenAddress(),
		Handler: nil,
		TLSConfig: &tls.Config {
			MinVersion: tls.VersionTLS12,
			CipherSuites: []uint16 {
				tls.TLS_RSA_WITH_AES_128_GCM_SHA256,
				tls.TLS_RSA_WITH_AES_256_GCM_SHA384,
				tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
				tls.TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256,
				tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
				tls.TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384,
				tls.TLS_ECDHE_RSA_WITH_CHACHA20_POLY1305_SHA256,
				tls.TLS_ECDHE_ECDSA_WITH_CHACHA20_POLY1305_SHA256,
				tls.TLS_AES_128_GCM_SHA256,
				tls.TLS_AES_256_GCM_SHA384,
				tls.TLS_CHACHA20_POLY1305_SHA256,
			},
		},
		ReadTimeout: c.GetReadTimeout(),
		MaxHeaderBytes: c.GetMaxHeaderBytes(),
	}

	log.Fatal(srv.ListenAndServeTLS(c.GetCertificatePath(), c.GetKeyPath()))
}
