package files

import (
	"bytes"
	"errors"
	"example.com/hackathon/apps/api/internal/auth"
	"example.com/hackathon/apps/api/internal/platform"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"io"
	"mime"
	"net/http"
	"strconv"
	"strings"
	"time"
)

const MaxSize = 10 << 20

type File struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Size        int       `json:"size"`
	ContentType string    `json:"content_type"`
	CreatedAt   time.Time `json:"created_at"`
}

func Register(group *gin.RouterGroup, d *platform.Dependencies) {
	group.GET("/files", func(c *gin.Context) {
		limit, err := strconv.Atoi(c.DefaultQuery("limit", "20"))
		offset, offsetErr := strconv.Atoi(c.DefaultQuery("offset", "0"))
		if err != nil || offsetErr != nil || limit < 1 || limit > 100 || offset < 0 || offset > 100000 {
			fail(c, 400, "limit должен быть от 1 до 100, offset — от 0 до 100000")
			return
		}
		page, err := List(c.Request.Context(), d.DB, auth.Current(c).Subject, limit, offset)
		if err != nil {
			_ = c.Error(err)
			fail(c, 500, "Не удалось загрузить список файлов")
			return
		}
		c.JSON(200, page)
	})
	group.POST("/files", func(c *gin.Context) {
		if err := c.Request.ParseMultipartForm(MaxSize + 1); err != nil {
			fail(c, 413, "Файл превышает 10 МиБ или запрос имеет неверный формат")
			return
		}
		defer c.Request.MultipartForm.RemoveAll()
		header, err := c.FormFile("file")
		if err != nil {
			fail(c, 400, "Выберите файл в поле file")
			return
		}
		input, err := header.Open()
		if err != nil {
			fail(c, 400, "Не удалось прочитать файл")
			return
		}
		defer input.Close()
		data, err := io.ReadAll(io.LimitReader(input, MaxSize+1))
		if err != nil || len(data) == 0 || len(data) > MaxSize {
			fail(c, 400, "Размер файла должен быть от 1 байта до 10 МиБ")
			return
		}
		contentType := http.DetectContentType(data)
		allowed := map[string]bool{"image/png": true, "image/jpeg": true, "image/webp": true, "application/pdf": true, "text/plain; charset=utf-8": true}
		if !allowed[contentType] {
			fail(c, 415, "Можно загрузить PNG, JPEG, WebP, PDF или текстовый файл")
			return
		}
		f := File{ID: uuid.NewString(), Name: header.Filename, Size: len(data), ContentType: contentType, CreatedAt: time.Now().UTC()}
		if len(f.Name) > 255 || strings.ContainsAny(f.Name, "\r\n") {
			fail(c, 400, "Недопустимое имя файла")
			return
		}
		key := auth.Current(c).Subject + "/" + f.ID
		_, err = d.Objects.PutObject(c.Request.Context(), &s3.PutObjectInput{Bucket: aws.String(d.Bucket), Key: aws.String(key), Body: bytes.NewReader(data), ContentType: aws.String(contentType)})
		if err != nil {
			_ = c.Error(err)
			fail(c, 503, "Хранилище временно недоступно")
			return
		}
		_, err = d.DB.Exec(c.Request.Context(), "INSERT INTO files(id,owner_id,object_key,name,size,content_type,created_at) VALUES($1,$2,$3,$4,$5,$6,$7)", f.ID, auth.Current(c).Subject, key, f.Name, f.Size, f.ContentType, f.CreatedAt)
		if err != nil {
			_, _ = d.Objects.DeleteObject(c.Request.Context(), &s3.DeleteObjectInput{Bucket: aws.String(d.Bucket), Key: aws.String(key)})
			_ = c.Error(err)
			fail(c, 500, "Не удалось сохранить файл")
			return
		}
		c.JSON(201, f)
	})
	group.GET("/files/:id", func(c *gin.Context) {
		if _, err := uuid.Parse(c.Param("id")); err != nil {
			fail(c, 400, "Ожидается идентификатор UUID")
			return
		}
		var key, name, contentType string
		err := d.DB.QueryRow(c.Request.Context(), "SELECT object_key,name,content_type FROM files WHERE id=$1 AND owner_id=$2", c.Param("id"), auth.Current(c).Subject).Scan(&key, &name, &contentType)
		if errors.Is(err, pgx.ErrNoRows) {
			fail(c, 404, "Файл не найден")
			return
		}
		if err != nil {
			_ = c.Error(err)
			fail(c, 500, "Не удалось загрузить файл")
			return
		}
		object, err := d.Objects.GetObject(c.Request.Context(), &s3.GetObjectInput{Bucket: aws.String(d.Bucket), Key: aws.String(key)})
		if err != nil {
			_ = c.Error(err)
			fail(c, 503, "Хранилище временно недоступно")
			return
		}
		defer object.Body.Close()
		c.Header("Content-Disposition", mime.FormatMediaType("attachment", map[string]string{"filename": name}))
		c.Header("X-Content-Type-Options", "nosniff")
		c.DataFromReader(200, aws.ToInt64(object.ContentLength), contentType, object.Body, nil)
	})
	group.DELETE("/files/:id", func(c *gin.Context) {
		if _, err := uuid.Parse(c.Param("id")); err != nil {
			fail(c, 400, "Ожидается идентификатор UUID")
			return
		}
		var key string
		err := d.DB.QueryRow(c.Request.Context(), "SELECT object_key FROM files WHERE id=$1 AND owner_id=$2", c.Param("id"), auth.Current(c).Subject).Scan(&key)
		if errors.Is(err, pgx.ErrNoRows) {
			fail(c, 404, "Файл не найден")
			return
		}
		if err != nil {
			_ = c.Error(err)
			fail(c, 500, "Не удалось загрузить файл")
			return
		}
		_, err = d.Objects.DeleteObject(c.Request.Context(), &s3.DeleteObjectInput{Bucket: aws.String(d.Bucket), Key: aws.String(key)})
		if err != nil {
			_ = c.Error(err)
			fail(c, 503, "Хранилище временно недоступно")
			return
		}
		if _, err = d.DB.Exec(c.Request.Context(), "DELETE FROM files WHERE id=$1 AND owner_id=$2", c.Param("id"), auth.Current(c).Subject); err != nil {
			_ = c.Error(err)
			fail(c, 500, "Не удалось удалить запись о файле")
			return
		}
		c.Status(204)
	})

}
func fail(c *gin.Context, status int, message string) {
	c.AbortWithStatusJSON(status, gin.H{"error": gin.H{"code": "file_error", "message": message, "request_id": c.GetString("request_id")}})
}
