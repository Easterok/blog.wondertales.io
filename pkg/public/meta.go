package public

import (
	"encoding/xml"
	"fmt"
	"strings"
	"time"

	"github.com/labstack/gommon/log"

	"github.com/easterok/blogs/pkg/db"
	"github.com/easterok/blogs/pkg/middlewares"
	"github.com/easterok/blogs/pkg/utils"
	"github.com/labstack/echo/v4"
)

type MetaApi struct {
	DB *db.DB
}

func NewMetaApi(db *db.DB) MetaApi {
	return MetaApi{
		DB: db,
	}
}

func (m *MetaApi) RobotsTxt(c echo.Context) error {
	host := middlewares.GetContextHost(c.Request().Context())

	c.Response().Header().Set(echo.HeaderContentType, echo.MIMETextPlainCharsetUTF8)
	c.Response().Writer.Write([]byte(fmt.Sprintf("User-agent: *\nDisallow:\nSitemap: %s/sitemap.xml", host)))

	return nil
}

func (m *MetaApi) Favicon(c echo.Context) error {
	return c.File("/static/icons/favicon.ico")
}

type sitemapCache struct {
	util time.Time

	ru utils.Sitemap
	en utils.Sitemap
}

type SitemapConfig struct {
	Accept []string

	InvalidateCache chan interface{}
}

func (m *MetaApi) Sitemmap(cfg SitemapConfig) func(c echo.Context) error {
	now := time.Now()

	cache := sitemapCache{
		util: now.Add(-time.Hour),
	}

	if cfg.InvalidateCache != nil {
		go func(c chan interface{}) {
			for range c {
				cache.util = time.Now().Add(-time.Hour)
			}
		}(cfg.InvalidateCache)
	}

	return func(c echo.Context) error {
		c.Response().Header().Set(echo.HeaderContentType, "application/xml")

		base := middlewares.GetContextHost(c.Request().Context())

		if time.Now().Before(cache.util) {
			c.Response().Header().Set("X-Sitemap-Cache", "hit")
			c.Response().Header().Set("X-Sitemap-Cache-Until", cache.util.Format(time.DateOnly))
			c.Response().Writer.Write(utils.XmlHeader)

			cached := cache.en

			if strings.HasPrefix(base, "https://blog.wonder-tales.ru") {
				cached = cache.ru
			}

			return xml.NewEncoder(c.Response().Writer).Encode(cached)
		}

		u := []utils.URL{
			{
				Loc:      "",
				Priority: "1",
				LastMod:  now.Format(time.DateOnly),
			},
			{
				Loc:      "articles",
				Priority: "0.9",
				LastMod:  now.Format(time.DateOnly),
			},
		}

		articles, err := m.DB.FindArticles(nil)
		if err != nil {
			log.Errorf("[Sitemap] Failed to load articles: %s\n", err.Error())
		} else {
			for _, article := range *articles {
				u = append(u, utils.URL{
					Loc:      fmt.Sprintf("article/%s", article.PathEng),
					LastMod:  article.UpdatedAt.Format(time.DateOnly),
					Priority: "0.9",
				}, utils.URL{
					Loc:      fmt.Sprintf("article/%s", article.Path),
					LastMod:  article.UpdatedAt.Format(time.DateOnly),
					Priority: "0.9",
				})
			}
		}

		stories, err := m.DB.FindStories(nil)
		if err != nil {
			log.Errorf("[Sitemap] Failed to load stories: %s\n", err.Error())
		} else {
			for _, story := range *stories {
				u = append(u, utils.URL{
					Loc:      fmt.Sprintf("story/%s", story.PathEng),
					LastMod:  story.UpdatedAt.Format(time.DateOnly),
					Priority: "0.9",
				}, utils.URL{
					Loc:      fmt.Sprintf("story/%s", story.Path),
					LastMod:  story.UpdatedAt.Format(time.DateOnly),
					Priority: "0.9",
				})
			}
		}

		articlesCategories, err := m.DB.FindNotHiddenCategories(db.ARTICLES_CATALOG)
		if err != nil {
			log.Errorf("[Sitemap] Failed to load categories for articles: %s\n", err.Error())
		} else {
			for _, story := range *articlesCategories {
				u = append(u, utils.URL{
					Loc:      fmt.Sprintf("a/%s", story.PathEng),
					LastMod:  story.UpdatedAt.Format(time.DateOnly),
					Priority: "0.8",
				}, utils.URL{
					Loc:      fmt.Sprintf("a/%s", story.Path),
					LastMod:  story.UpdatedAt.Format(time.DateOnly),
					Priority: "0.8",
				})
			}
		}

		storyCategories, err := m.DB.FindNotHiddenCategories(db.TALES_CATALOG)
		if err != nil {
			log.Errorf("[Sitemap] Failed to load categories for story: %s\n", err.Error())
		} else {
			for _, story := range *storyCategories {
				u = append(u, utils.URL{
					Loc:      fmt.Sprintf("s/%s", story.PathEng),
					LastMod:  story.UpdatedAt.Format(time.DateOnly),
					Priority: "0.8",
				}, utils.URL{
					Loc:      fmt.Sprintf("s/%s", story.Path),
					LastMod:  story.UpdatedAt.Format(time.DateOnly),
					Priority: "0.8",
				})
			}
		}

		cache.util = time.Now().Add(time.Hour * 24 * 365)
		cache.en = utils.NewSitemap("https://blog.wondertales.io", u, cfg.Accept)
		cache.ru = utils.NewSitemap("https://blog.wonder-tales.ru", u, cfg.Accept)

		cached := cache.en

		if strings.HasPrefix(base, "https://blog.wonder-tales.ru") {
			cached = cache.ru
		}

		c.Response().Writer.Write(utils.XmlHeader)

		return xml.NewEncoder(c.Response().Writer).Encode(cached)
	}
}
