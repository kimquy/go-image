package main

import (
	"io"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/otiai10/gosseract"
)

func stringProcessing(s string) string {
	r := strings.Replace(s, " ", "", -1)
	r = strings.TrimSpace(r)
	r = strings.TrimPrefix(r, ".")
	r = strings.TrimSuffix(r, "%")
	return r
}

func imageProcessing(c *gin.Context) {
	client, _ := gosseract.NewClient()
	file, header, err := c.Request.FormFile("file")
	file_path := "./tmp/" + header.Filename
	image, err := os.Create(file_path)
	if err != nil {
		log.Fatal(err)
	}
	defer image.Close()
	_, err = io.Copy(image, file)
	if err != nil {
		log.Fatal(err)
	}

	out, _ := client.Src(file_path).Out()
	out = stringProcessing(out)

	c.String(http.StatusOK, out)
}

func home(c *gin.Context) {
	c.String(http.StatusOK, "Ok")
}

func main() {
	router := gin.Default()
	router.POST("/post_image", imageProcessing)
	router.GET("/", home)
	router.Run(":8080")
}
