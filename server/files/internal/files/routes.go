package files

import (
	"errors"
	"io"
	"mime"

	"example.com/hackathon/server/files/internal/platform"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// Register добавляет маршруты /files: список (фильтр entity_type+entity_id), загрузка multipart
// (поле file, необязательные entity_type и entity_id), скачивание и удаление по id.
func Register(group *gin.RouterGroup, s *Service) {
	group.GET("/files", func(c *gin.Context) {
		limit, offset, ok := platform.ParsePagination(c.DefaultQuery("limit", "20"), c.DefaultQuery("offset", "0"))
		if !ok {
			fail(c, 400, "invalid_pagination", "limit должен быть от 1 до 100, offset — от 0 до 100000")
			return
		}
		entity, err := ParseEntity(c.Query("entity_type"), c.Query("entity_id"))
		if err != nil {
			fail(c, 400, "invalid_entity", err.Error())
			return
		}
		page, err := s.List(c.Request.Context(), platform.Owner(c), entity, limit, offset)
		if err != nil {
			_ = c.Error(err)
			fail(c, 500, "internal_error", "Не удалось загрузить список файлов")
			return
		}
		c.JSON(200, page)
	})
	group.POST("/files", func(c *gin.Context) {
		if err := c.Request.ParseMultipartForm(MaxSize + 1); err != nil {
			fail(c, 413, "payload_too_large", "Файл превышает 10 МиБ или запрос имеет неверный формат")
			return
		}
		defer c.Request.MultipartForm.RemoveAll()
		header, err := c.FormFile("file")
		if err != nil {
			fail(c, 400, "invalid_body", "Выберите файл в поле file")
			return
		}
		input, err := header.Open()
		if err != nil {
			fail(c, 400, "invalid_body", "Не удалось прочитать файл")
			return
		}
		defer input.Close()
		data, err := io.ReadAll(io.LimitReader(input, MaxSize+1))
		if err != nil {
			fail(c, 400, "invalid_body", "Не удалось прочитать файл")
			return
		}
		entity, err := ParseEntity(c.Request.FormValue("entity_type"), c.Request.FormValue("entity_id"))
		if err != nil {
			fail(c, 400, "invalid_entity", err.Error())
			return
		}
		f, err := s.UploadFor(c.Request.Context(), platform.Owner(c), header.Filename, data, entity)
		if handleUploadError(c, err) {
			return
		}
		c.JSON(201, f)
	})
	group.GET("/files/:id", func(c *gin.Context) {
		if _, err := uuid.Parse(c.Param("id")); err != nil {
			fail(c, 400, "invalid_id", "Ожидается идентификатор UUID")
			return
		}
		body, size, name, contentType, err := s.Download(c.Request.Context(), platform.Owner(c), c.Param("id"))
		if handleFileError(c, err) {
			return
		}
		defer body.Close()
		c.Header("Content-Disposition", mime.FormatMediaType("attachment", map[string]string{"filename": name}))
		c.Header("X-Content-Type-Options", "nosniff")
		c.DataFromReader(200, size, contentType, body, nil)
	})
	group.DELETE("/files/:id", func(c *gin.Context) {
		if _, err := uuid.Parse(c.Param("id")); err != nil {
			fail(c, 400, "invalid_id", "Ожидается идентификатор UUID")
			return
		}
		if handleFileError(c, s.Delete(c.Request.Context(), platform.Owner(c), c.Param("id"))) {
			return
		}
		c.Status(204)
	})
}

func handleUploadError(c *gin.Context, err error) bool {
	switch {
	case err == nil:
		return false
	case errors.Is(err, ErrInvalidSize), errors.Is(err, ErrInvalidName):
		fail(c, 400, "invalid_body", err.Error())
	case errors.Is(err, ErrUnsupportedType):
		fail(c, 415, "unsupported_type", err.Error())
	case errors.Is(err, ErrStorageUnavailable):
		_ = c.Error(err)
		fail(c, 503, "storage_unavailable", "Хранилище временно недоступно")
	default:
		_ = c.Error(err)
		fail(c, 500, "internal_error", "Не удалось сохранить файл")
	}
	return true
}

func handleFileError(c *gin.Context, err error) bool {
	switch {
	case err == nil:
		return false
	case errors.Is(err, ErrNotFound):
		fail(c, 404, "not_found", "Файл не найден")
	case errors.Is(err, ErrStorageUnavailable):
		_ = c.Error(err)
		fail(c, 503, "storage_unavailable", "Хранилище временно недоступно")
	default:
		_ = c.Error(err)
		fail(c, 500, "internal_error", "Не удалось обработать файл")
	}
	return true
}

func fail(c *gin.Context, status int, code, message string) {
	c.AbortWithStatusJSON(status, gin.H{"error": gin.H{"code": code, "message": message, "request_id": c.GetString("request_id")}})
}
