package main

import (
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/easterok/blogs/pkg/db"
	"github.com/easterok/blogs/pkg/middlewares"
	"github.com/easterok/blogs/pkg/public"
	"github.com/easterok/blogs/pkg/s3"
	"github.com/easterok/blogs/pkg/utils"
	views "github.com/easterok/blogs/pkg/views"
	"github.com/go-playground/validator"
	"github.com/joho/godotenv"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/labstack/gommon/log"
	"golang.org/x/net/context"
	"gorm.io/gorm"
)

type AppEnv struct {
	ApiPort string

	DBDns string

	AwsRegion string
	AwsBucket string

	AdminEmail    string
	AdminPassword string
}

func parseEnv() (*AppEnv, error) {
	if os.Getenv("LOCAL") == "1" {
		err := godotenv.Load()

		if err != nil {
			return nil, fmt.Errorf("error loading .env file")
		}
	}

	return &AppEnv{
		ApiPort:   os.Getenv("API_PORT"),
		DBDns:     os.Getenv("PG_DNS"),
		AwsBucket: os.Getenv("AWS_BUCKET"),
		AwsRegion: os.Getenv("AWS_REGION"),

		AdminEmail:    os.Getenv("ADMIN_ACCESS_EMAIL"),
		AdminPassword: os.Getenv("ADMIN_ACCESS_PASSWORD"),
	}, nil
}

type CustomValidator struct {
	validator *validator.Validate
}

func (cv *CustomValidator) Validate(i interface{}) error {
	if err := cv.validator.Struct(i); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}
	return nil
}

var ACCEPT_LANGUAGES = []string{"en", "ru"}

func main() {
	env, err := parseEnv()

	if err != nil {
		log.Fatal(err)
	}

	database, err := db.Connect(env.DBDns, &gorm.Config{
		// Logger: logger.Default.LogMode(logger.Info),
	})

	if err != nil {
		log.Fatal(err)
	}

	awsConfig, err := config.LoadDefaultConfig(context.Background())

	if err != nil {
		log.Fatal(err)
	}

	s3Client := s3.NewClient(awsConfig)

	e := echo.New()

	now := time.Now()
	hash := fmt.Sprintf("%d", now.UnixNano()/1e6)

	// e.Logger.SetLevel(log.WARN)
	e.Use(middleware.Logger())
	e.Use(middleware.Gzip())
	e.Use(middleware.RequestID())
	e.Use(middlewares.StaticCache)
	e.Use(middlewares.EtagWithConfig(middlewares.EtagConfig{
		Skipper: func(c echo.Context) bool {
			return c.Request().URL.Query().Has("q")
		},
	}))
	e.Pre(middleware.RemoveTrailingSlashWithConfig(middleware.TrailingSlashConfig{
		RedirectCode: http.StatusPermanentRedirect,
	}))
	e.Use(middleware.SecureWithConfig(
		middleware.SecureConfig{
			HSTSMaxAge:         31536000,
			XFrameOptions:      "SAMEORIGIN",
			ContentTypeNosniff: "nosniff",
			XSSProtection:      "1; mode=block",
		},
	))
	e.Use(middlewares.StaticHashWithConfig(middlewares.StaticHashConfig{
		Hash: hash,
	}))
	e.Use(middlewares.PreferLanguageWithConfig(middlewares.PreferLanguageConfig{
		Accept:   ACCEPT_LANGUAGES,
		Fallback: ACCEPT_LANGUAGES[0],
	}))
	middlewares.ServiceStaticWithHash(e, []string{"static/admin/css/index.css", "static/index.css"}, hash)

	e.Static("/static", "static")

	e.Validator = &CustomValidator{validator: validator.New()}

	pub := public.NewApi(database)
	public_meta := public.NewMetaApi(database)

	invalidateCacheChan := make(chan interface{}, 1)

	var invalidateSitemapCache func()
	invalidateSitemapCache = func() {
		invalidateCacheChan <- 1
	}

	e.GET("/favicon.ico", public_meta.Favicon)
	e.GET("/robots.txt", public_meta.RobotsTxt)
	e.GET("/sitemap.xml", public_meta.Sitemmap(public.SitemapConfig{
		Accept:          ACCEPT_LANGUAGES,
		InvalidateCache: invalidateCacheChan,
	}))

	e.GET("/", pub.Home)
	e.GET("/s/:path", pub.CategoryPage)
	e.GET("/story/:path", pub.StoryPage)
	e.GET("/a/:path", pub.ArticleCategoryPage)
	e.GET("/articles", pub.ArticlesPage)
	e.GET("/article/:path", pub.ArticlePage)

	for _, lang := range ACCEPT_LANGUAGES {
		group := e.Group("/" + lang)
		group.GET("", pub.Home)
		group.GET("/s/:path", pub.CategoryPage)
		group.GET("/story/:path", pub.StoryPage)
		group.GET("/a/:path", pub.ArticleCategoryPage)
		group.GET("/articles", pub.ArticlesPage)
		group.GET("/article/:path", pub.ArticlePage)
	}

	// e.GET("*", func(c echo.Context) error {
	// 	return views.P404().Render(c.Request().Context(), c.Response().Writer)
	// })

	adminGr := e.Group("/admin")

	adminGr.GET("", func(c echo.Context) error {
		stories, err := database.FindAdminStories()

		if err != nil {
			return c.String(http.StatusOK, err.Error())
		}

		props := views.AdminTalesProps{
			Tales: stories,
		}

		return views.AdminTales(props).Render(c.Request().Context(), c.Response().Writer)
	})
	adminGr.GET("/tales", func(c echo.Context) error {
		stories, err := database.FindAdminStories()

		if err != nil {
			return c.String(http.StatusOK, err.Error())
		}

		props := views.AdminTalesProps{
			Tales: stories,
		}

		return views.AdminTales(props).Render(c.Request().Context(), c.Response().Writer)
	})
	adminGr.GET("/tales/catalog", func(c echo.Context) error {
		cat := database.GetAllCatalogItems(db.TALES_CATALOG)

		props := views.AdminCatalogProps{
			Name:     "сказок",
			Items:    cat,
			EditLink: "/admin/tales/catalog",
			Link:     "/s",
		}

		return views.AdminCatalog(props).Render(c.Request().Context(), c.Response().Writer)
	})
	adminGr.GET("/tales/:id", func(c echo.Context) error {
		id, err := strconv.ParseUint(c.Param("id"), 10, 32)

		if err != nil {
			return err
		}

		story, err := database.GetStoryById(uint(id))

		if err != nil {
			return c.String(http.StatusBadRequest, err.Error())
		}

		catalog := database.GetAllCatalogItems(db.TALES_CATALOG)

		props := views.EditTaleProps{
			CatalogItems: catalog,
			Story:        story,
		}

		return views.EditTale(props).Render(c.Request().Context(), c.Response().Writer)
	})
	adminGr.DELETE("/tales/:id", func(c echo.Context) error {
		id, err := strconv.ParseUint(c.Param("id"), 10, 32)

		if err != nil {
			return err
		}

		err = database.DeleteStory(uint(id))

		if err != nil {
			return c.String(http.StatusBadRequest, fmt.Sprintf("Failed to delete story: %s", err.Error()))
		}

		c.Response().Header().Set("HX-Redirect", "/admin/tales")
		invalidateSitemapCache()

		return c.String(http.StatusOK, "OK")
	})
	type NewTale struct {
		CategoryId string `form:"categoryId"`
	}
	adminGr.POST("/tales/new", func(c echo.Context) error {
		r := new(NewTale)

		if err := c.Bind(r); err != nil {
			return c.String(http.StatusBadRequest, err.Error())
		}

		if err = c.Validate(r); err != nil {
			return c.String(http.StatusBadRequest, err.Error())
		}

		draft, err := database.CreateDraftStory(r.CategoryId)

		if err != nil {
			return err
		}

		c.Response().Header().Set("HX-Redirect", fmt.Sprintf("/admin/tales/%d", draft.ID))
		invalidateSitemapCache()

		return c.String(http.StatusOK, "OK")
	})
	adminGr.POST("/tales/catalog/new", func(c echo.Context) error {
		n := db.Catalog{
			Type:    db.TALES_CATALOG,
			Name:    "Новый раздел",
			NameEng: "New catalog",
			Hidden:  db.ToCheckboxValue("on"),
		}

		err := database.CreateCatalog(&n)

		if err != nil {
			return err
		}

		c.Response().Header().Set("HX-Redirect", fmt.Sprintf("/admin/tales/catalog/%d", n.ID))
		invalidateSitemapCache()

		return c.String(http.StatusOK, "OK")
	})
	adminGr.GET("/tales/catalog/:id", func(c echo.Context) error {
		id, err := strconv.ParseUint(c.Param("id"), 10, 32)

		if err != nil {
			return err
		}

		category, err := database.FindCategoryById(uint(id))

		if err != nil {
			return c.String(http.StatusBadRequest, err.Error())
		}

		props := views.AdminCategoryPageProps{
			Category: category,
		}

		return views.AdminCategoryPage(props).Render(c.Request().Context(), c.Response().Writer)
	})
	adminGr.POST("/tales/:id/cover", func(c echo.Context) error {
		id, err := strconv.ParseUint(c.Param("id"), 10, 32)

		if err != nil {
			return c.String(http.StatusBadRequest, err.Error())
		}

		fileHeader, err := utils.ValidateFile(c, "file", 10<<20)
		if err != nil {
			return err
		}
		ctx, close := context.WithTimeout(context.Background(), time.Minute)
		defer close()

		fileKey := "blog/" + utils.UUIDFileName(fileHeader.Filename)

		f, err := fileHeader.Open()
		if err != nil {
			return c.String(http.StatusBadRequest, err.Error())
		}
		defer f.Close()

		awsUrl, err := s3Client.UploadFile(ctx, env.AwsBucket, fileKey, f)
		if err != nil {
			return c.String(http.StatusBadRequest, err.Error())
		}

		err = database.UpdateTale(uint(id), &db.Story{
			Cover: awsUrl,
		})
		if err != nil {
			return c.String(http.StatusBadRequest, err.Error())
		}

		invalidateSitemapCache()

		return c.String(http.StatusOK, awsUrl)
	})
	type PatchTale struct {
		Name    string `form:"name"`
		NameEng string `form:"nameEng"`

		Prefix    string `form:"prefix"`
		PrefixEng string `form:"prefixEng"`

		SeoDesc        string `form:"seoDesc"`
		SeoDescEng     string `form:"seoDescEng"`
		SeoKeywords    string `form:"seoKeywords"`
		SeoKeywordsEng string `form:"seoKeywordsEng"`

		Postfix    string `form:"postfix"`
		PostfixEng string `form:"postfixEng"`

		Published string `form:"published"`
	}
	adminGr.PATCH("/tales/:id", func(c echo.Context) error {
		id, err := strconv.ParseUint(c.Param("id"), 10, 32)

		if err != nil {
			return c.String(http.StatusBadRequest, err.Error())
		}

		r := new(PatchTale)

		if err := c.Bind(r); err != nil {
			return c.String(http.StatusBadRequest, err.Error())
		}

		if err = c.Validate(r); err != nil {
			return c.String(http.StatusBadRequest, err.Error())
		}

		storyToUpdate := db.Story{
			Name:    strings.TrimSpace(r.Name),
			NameEng: strings.TrimSpace(r.NameEng),

			Prefix:    strings.TrimSpace(r.Prefix),
			PrefixEng: strings.TrimSpace(r.PrefixEng),

			Postfix:        strings.TrimSpace(r.Postfix),
			PostfixEng:     strings.TrimSpace(r.PostfixEng),
			SeoDesc:        strings.TrimSpace(r.SeoDesc),
			SeoDescEng:     strings.TrimSpace(r.SeoDescEng),
			SeoKeywords:    strings.TrimSpace(r.SeoKeywords),
			SeoKeywordsEng: strings.TrimSpace(r.SeoKeywordsEng),

			Published: db.ToCheckboxValue(r.Published),
		}

		err = database.UpdateTale(uint(id), &storyToUpdate)

		if err != nil {
			return c.String(http.StatusBadRequest, err.Error())
		}

		c.Response().Header().Set("HX-RETARGET", "#updateResult")
		c.Response().Header().Set("HX-RESWAP", "afterbegin")

		invalidateSitemapCache()

		return views.LastUpdate(time.Now()).Render(c.Request().Context(), c.Response().Writer)
	})
	type PatchConnection struct {
		CatalogId int `form:"catalog"`
	}
	adminGr.PATCH("/tale/:taleId/connection/:id", func(c echo.Context) error {
		id, err := strconv.ParseUint(c.Param("id"), 10, 32)

		if err != nil {
			return c.String(http.StatusBadRequest, err.Error())
		}

		taleId, err := strconv.ParseUint(c.Param("taleId"), 10, 32)

		if err != nil {
			return c.String(http.StatusBadRequest, err.Error())
		}

		r := new(PatchConnection)

		if err := c.Bind(r); err != nil {
			return c.String(http.StatusBadRequest, err.Error())
		}

		if r.CatalogId == -1 {
			if err = database.DeleteStoryConnection(uint(id)); err != nil {
				return c.String(http.StatusBadRequest, err.Error())
			}
		} else {
			if err = database.UpdateStoryConnection(uint(id), uint(r.CatalogId)); err != nil {
				return c.String(http.StatusBadRequest, err.Error())
			}
		}

		invalidateSitemapCache()

		categories, err := database.FindCatalogStoriesByStoryId(uint(taleId))
		if err != nil {
			return c.String(http.StatusBadRequest, "Категория обновлена, но не смогли загрузить обновленные")
		}

		catalog := database.GetAllCatalogItems(db.TALES_CATALOG)

		return views.StoryConnections(*categories, uint(taleId), catalog).Render(c.Request().Context(), c.Response().Writer)
	})
	type CreateConnection struct {
		CatalogId uint `form:"catalog"`
	}
	adminGr.POST("/tale/:taleId/connection", func(c echo.Context) error {
		taleId, err := strconv.ParseUint(c.Param("taleId"), 10, 32)

		if err != nil {
			return c.String(http.StatusBadRequest, err.Error())
		}

		r := new(CreateConnection)

		if err := c.Bind(r); err != nil {
			return c.String(http.StatusBadRequest, err.Error())
		}

		_, err = database.CreateStoryConnection(uint(taleId), r.CatalogId)

		if err != nil {
			return c.String(http.StatusBadRequest, err.Error())
		}

		invalidateSitemapCache()

		categories, err := database.FindCatalogStoriesByStoryId(uint(taleId))
		if err != nil {
			return c.String(http.StatusBadRequest, "Категория добавлена, но не смогли загрузить обновленные")
		}

		catalog := database.GetAllCatalogItems(db.TALES_CATALOG)

		return views.StoryConnections(*categories, uint(taleId), catalog).Render(c.Request().Context(), c.Response().Writer)
	})
	adminGr.GET("/tales/:id/validate", func(c echo.Context) error {
		id, err := strconv.ParseUint(c.Param("id"), 10, 32)

		if err != nil {
			return c.String(http.StatusBadRequest, err.Error())
		}

		story, err := database.GetStoryById(uint(id))

		if err != nil {
			return c.String(http.StatusBadRequest, err.Error())
		}

		return views.ValidateTale(story).Render(c.Request().Context(), c.Response().Writer)
	})
	adminGr.GET("/category/:id/validate", func(c echo.Context) error {
		id, err := strconv.ParseUint(c.Param("id"), 10, 32)

		if err != nil {
			return c.String(http.StatusBadRequest, err.Error())
		}

		category, err := database.FindCategoryById(uint(id))

		if err != nil {
			return c.String(http.StatusBadRequest, err.Error())
		}

		return views.ValidateCategory(category).Render(c.Request().Context(), c.Response().Writer)
	})
	adminGr.GET("/article/:id/validate", func(c echo.Context) error {
		id, err := strconv.ParseUint(c.Param("id"), 10, 32)

		if err != nil {
			return c.String(http.StatusBadRequest, err.Error())
		}

		article, err := database.GetArticleById(uint(id))

		if err != nil {
			return c.String(http.StatusBadRequest, err.Error())
		}

		return views.ValidateArticle(article).Render(c.Request().Context(), c.Response().Writer)
	})

	adminGr.DELETE("/category/:id", func(c echo.Context) error {
		id, err := strconv.ParseUint(c.Param("id"), 10, 32)

		if err != nil {
			return err
		}

		err = database.DeleteCategory(uint(id))

		if err != nil {
			return c.String(http.StatusBadRequest, fmt.Sprintf("Ошибка удаления категории: %s", err.Error()))
		}

		c.Response().Header().Set("HX-Redirect", "/admin/tales/catalog")
		invalidateSitemapCache()

		return c.String(http.StatusOK, "OK")
	})
	adminGr.POST("/category/:id/cover", func(c echo.Context) error {
		id, err := strconv.ParseUint(c.Param("id"), 10, 32)

		if err != nil {
			return c.String(http.StatusBadRequest, err.Error())
		}

		fileHeader, err := utils.ValidateFile(c, "file", 10<<20)
		if err != nil {
			return err
		}
		ctx, close := context.WithTimeout(context.Background(), time.Minute)
		defer close()

		fileKey := "blog/" + utils.UUIDFileName(fileHeader.Filename)

		f, err := fileHeader.Open()
		if err != nil {
			return c.String(http.StatusBadRequest, err.Error())
		}
		defer f.Close()

		awsUrl, err := s3Client.UploadFile(ctx, env.AwsBucket, fileKey, f)
		if err != nil {
			return c.String(http.StatusBadRequest, err.Error())
		}

		err = database.UpdateCatalog(uint(id), &db.Catalog{
			Cover: awsUrl,
		})
		if err != nil {
			return c.String(http.StatusBadRequest, err.Error())
		}
		invalidateSitemapCache()

		return c.String(http.StatusOK, awsUrl)
	})
	type PatchCategory struct {
		Name    string `form:"name"`
		NameEng string `form:"nameEng"`

		SeoDesc        string `form:"seoDesc"`
		SeoDescEng     string `form:"seoDescEng"`
		SeoKeywords    string `form:"seoKeywords"`
		SeoKeywordsEng string `form:"seoKeywordsEng"`

		Hidden     string `form:"hidden"`
		ShowOnMain string `form:"showOnMain"`
	}
	adminGr.PATCH("/category/:id", func(c echo.Context) error {
		id, err := strconv.ParseUint(c.Param("id"), 10, 32)

		if err != nil {
			return c.String(http.StatusBadRequest, err.Error())
		}

		r := new(PatchCategory)

		if err := c.Bind(r); err != nil {
			return c.String(http.StatusBadRequest, err.Error())
		}

		if err = c.Validate(r); err != nil {
			return c.String(http.StatusBadRequest, err.Error())
		}

		catalogToUpdate := db.Catalog{
			Name:    r.Name,
			NameEng: r.NameEng,

			SeoDesc:        strings.TrimSpace(r.SeoDesc),
			SeoDescEng:     strings.TrimSpace(r.SeoDescEng),
			SeoKeywords:    strings.TrimSpace(r.SeoKeywords),
			SeoKeywordsEng: strings.TrimSpace(r.SeoKeywordsEng),

			Hidden:     db.ToCheckboxValue(r.Hidden),
			ShowOnMain: db.ToCheckboxValue(r.ShowOnMain),
		}

		err = database.UpdateCatalog(uint(id), &catalogToUpdate)

		if err != nil {
			return c.String(http.StatusBadRequest, err.Error())
		}

		c.Response().Header().Set("HX-RETARGET", "#updateResult")
		c.Response().Header().Set("HX-RESWAP", "afterbegin")
		invalidateSitemapCache()

		return views.LastUpdate(time.Now()).Render(c.Request().Context(), c.Response().Writer)
	})
	adminGr.GET("/articles/catalog", func(c echo.Context) error {
		cat := database.GetAllCatalogItems(db.ARTICLES_CATALOG)

		props := views.AdminCatalogProps{
			Name:     "статей",
			Items:    cat,
			EditLink: "/admin/articles/catalog",
			Link:     "/a",
		}

		return views.AdminCatalog(props).Render(c.Request().Context(), c.Response().Writer)
	})
	adminGr.POST("/articles/catalog/new", func(c echo.Context) error {
		n := db.Catalog{
			Type:    db.ARTICLES_CATALOG,
			Name:    "Новый раздел",
			NameEng: "New catalog",
			Hidden:  db.ToCheckboxValue("on"),
		}

		err := database.CreateCatalog(&n)

		if err != nil {
			return err
		}

		c.Response().Header().Set("HX-Redirect", fmt.Sprintf("/admin/articles/catalog/%d", n.ID))
		invalidateSitemapCache()

		return c.String(http.StatusOK, "OK")
	})
	adminGr.GET("/articles/catalog/:id", func(c echo.Context) error {
		id, err := strconv.ParseUint(c.Param("id"), 10, 32)

		if err != nil {
			return err
		}

		category, err := database.FindCategoryById(uint(id))

		if err != nil {
			return c.String(http.StatusBadRequest, err.Error())
		}

		props := views.AdminArticlesCategoryPageProps{
			Category: category,
		}

		return views.AdminArticlesCategoryPage(props).Render(c.Request().Context(), c.Response().Writer)
	})
	type NewArticle struct {
		CategoryId string `form:"categoryId"`
	}
	adminGr.POST("/articles/new", func(c echo.Context) error {
		r := new(NewArticle)

		if err := c.Bind(r); err != nil {
			return c.String(http.StatusBadRequest, err.Error())
		}

		if err = c.Validate(r); err != nil {
			return c.String(http.StatusBadRequest, err.Error())
		}

		draft, err := database.CreateDraftArticle(r.CategoryId)

		if err != nil {
			return err
		}

		c.Response().Header().Set("HX-Redirect", fmt.Sprintf("/admin/articles/%d", draft.ID))
		invalidateSitemapCache()

		return c.String(http.StatusOK, "OK")
	})
	adminGr.GET("/articles", func(c echo.Context) error {
		articles, err := database.FindAdminArticles()

		if err != nil {
			return c.String(http.StatusOK, err.Error())
		}

		props := views.AdminArticlesProps{
			Articles: articles,
		}

		return views.AdminArticles(props).Render(c.Request().Context(), c.Response().Writer)
	})
	adminGr.GET("/articles/:id", func(c echo.Context) error {
		id, err := strconv.ParseUint(c.Param("id"), 10, 32)

		if err != nil {
			return err
		}

		article, err := database.GetArticleById(uint(id))

		if err != nil {
			return c.String(http.StatusBadRequest, err.Error())
		}

		catalog := database.GetAllCatalogItems(db.ARTICLES_CATALOG)

		props := views.EditArticleProps{
			CatalogItems: catalog,
			Article:      article,
		}

		return views.EditArticle(props).Render(c.Request().Context(), c.Response().Writer)
	})

	adminGr.POST("/articles/:id/cover", func(c echo.Context) error {
		id, err := strconv.ParseUint(c.Param("id"), 10, 32)

		if err != nil {
			return c.String(http.StatusBadRequest, err.Error())
		}

		fileHeader, err := utils.ValidateFile(c, "file", 10<<20)
		if err != nil {
			return err
		}
		ctx, close := context.WithTimeout(context.Background(), time.Minute)
		defer close()

		fileKey := "blog/" + utils.UUIDFileName(fileHeader.Filename)

		f, err := fileHeader.Open()
		if err != nil {
			return c.String(http.StatusBadRequest, err.Error())
		}
		defer f.Close()

		awsUrl, err := s3Client.UploadFile(ctx, env.AwsBucket, fileKey, f)
		if err != nil {
			return c.String(http.StatusBadRequest, err.Error())
		}

		err = database.UpdateArticle(uint(id), &db.Article{
			Cover: awsUrl,
		})
		if err != nil {
			return c.String(http.StatusBadRequest, err.Error())
		}
		invalidateSitemapCache()

		return c.String(http.StatusOK, awsUrl)
	})
	adminGr.DELETE("/articles/:id", func(c echo.Context) error {
		id, err := strconv.ParseUint(c.Param("id"), 10, 32)

		if err != nil {
			return err
		}

		err = database.DeleteArticle(uint(id))

		if err != nil {
			return c.String(http.StatusBadRequest, fmt.Sprintf("Failed to delete article: %s", err.Error()))
		}

		c.Response().Header().Set("HX-Redirect", "/admin/articles")
		invalidateSitemapCache()

		return c.String(http.StatusOK, "OK")
	})
	type PatchArticle struct {
		Name    string `form:"name"`
		NameEng string `form:"nameEng"`

		SeoDesc        string `form:"seoDesc"`
		SeoDescEng     string `form:"seoDescEng"`
		SeoKeywords    string `form:"seoKeywords"`
		SeoKeywordsEng string `form:"seoKeywordsEng"`

		Prefix    string `form:"prefix"`
		PrefixEng string `form:"prefixEng"`

		Postfix    string `form:"postfix"`
		PostfixEng string `form:"postfixEng"`

		Published string `form:"published"`
	}
	adminGr.PATCH("/articles/:id", func(c echo.Context) error {
		id, err := strconv.ParseUint(c.Param("id"), 10, 32)

		if err != nil {
			return c.String(http.StatusBadRequest, err.Error())
		}

		r := new(PatchArticle)

		if err := c.Bind(r); err != nil {
			return c.String(http.StatusBadRequest, err.Error())
		}

		if err = c.Validate(r); err != nil {
			return c.String(http.StatusBadRequest, err.Error())
		}

		articleToUpdate := db.Article{
			Name:    strings.TrimSpace(r.Name),
			NameEng: strings.TrimSpace(r.NameEng),

			SeoDesc:        strings.TrimSpace(r.SeoDesc),
			SeoDescEng:     strings.TrimSpace(r.SeoDescEng),
			SeoKeywords:    strings.TrimSpace(r.SeoKeywords),
			SeoKeywordsEng: strings.TrimSpace(r.SeoKeywordsEng),

			Prefix:    strings.TrimSpace(r.Prefix),
			PrefixEng: strings.TrimSpace(r.PrefixEng),

			Postfix:    strings.TrimSpace(r.Postfix),
			PostfixEng: strings.TrimSpace(r.PostfixEng),

			Published: db.ToCheckboxValue(r.Published),
		}

		err = database.UpdateArticle(uint(id), &articleToUpdate)

		if err != nil {
			return c.String(http.StatusBadRequest, err.Error())
		}

		c.Response().Header().Set("HX-RETARGET", "#updateResult")
		c.Response().Header().Set("HX-RESWAP", "afterbegin")
		invalidateSitemapCache()

		return views.LastUpdate(time.Now()).Render(c.Request().Context(), c.Response().Writer)
	})
	adminGr.PATCH("/article/:articleId/connection/:id", func(c echo.Context) error {
		id, err := strconv.ParseUint(c.Param("id"), 10, 32)

		if err != nil {
			return c.String(http.StatusBadRequest, err.Error())
		}

		articleId, err := strconv.ParseUint(c.Param("articleId"), 10, 32)

		if err != nil {
			return c.String(http.StatusBadRequest, err.Error())
		}

		r := new(PatchConnection)

		if err := c.Bind(r); err != nil {
			return c.String(http.StatusBadRequest, err.Error())
		}

		if r.CatalogId == -1 {
			if err = database.DeleteArticleConnection(uint(id)); err != nil {
				return c.String(http.StatusBadRequest, err.Error())
			}
		} else {
			if err = database.UpdateArticleConnection(uint(id), uint(r.CatalogId)); err != nil {
				return c.String(http.StatusBadRequest, err.Error())
			}
		}

		categories, err := database.FindCatalogArticlesByArticleId(uint(articleId))
		if err != nil {
			return c.String(http.StatusBadRequest, "Категория обновлена, но не смогли загрузить обновленные")
		}

		catalog := database.GetAllCatalogItems(db.ARTICLES_CATALOG)

		return views.ArticleConnections(*categories, uint(articleId), catalog).Render(c.Request().Context(), c.Response().Writer)
	})
	adminGr.POST("/article/:articleId/connection", func(c echo.Context) error {
		articleId, err := strconv.ParseUint(c.Param("articleId"), 10, 32)

		if err != nil {
			return c.String(http.StatusBadRequest, err.Error())
		}

		r := new(CreateConnection)

		if err := c.Bind(r); err != nil {
			return c.String(http.StatusBadRequest, err.Error())
		}

		_, err = database.CreateArticleConnection(uint(articleId), r.CatalogId)

		if err != nil {
			return c.String(http.StatusBadRequest, err.Error())
		}

		categories, err := database.FindCatalogArticlesByArticleId(uint(articleId))
		if err != nil {
			return c.String(http.StatusBadRequest, "Категория добавлена, но не смогли загрузить обновленные")
		}

		catalog := database.GetAllCatalogItems(db.ARTICLES_CATALOG)

		return views.ArticleConnections(*categories, uint(articleId), catalog).Render(c.Request().Context(), c.Response().Writer)
	})

	e.Logger.Fatal(e.Start(fmt.Sprintf(":%s", env.ApiPort)))
}
