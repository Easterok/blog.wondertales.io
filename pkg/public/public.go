package public

import (
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/easterok/blogs/pkg/db"
	"github.com/easterok/blogs/pkg/utils"
	"github.com/easterok/blogs/pkg/views"
	"github.com/labstack/echo/v4"
)

type PublicApi struct {
	DB *db.DB
}

func NewApi(db *db.DB) PublicApi {
	return PublicApi{
		DB: db,
	}
}

func validatePath(s string) error {
	if len(s) < 1 || len(s) > 255 {
		return fmt.Errorf("not found")
	}

	return nil
}

var created, _ = time.Parse(time.DateTime, "2024-04-08 01:00:00")

var staticHomePageProps = views.BaseViewProps{
	Title:      "Читайте любимые сказки | WonderTales Blog",
	TitleEn:    "Read your favourite fairy tales | WonderTales Blog",
	Desc:       "Читайте любимые сказки вместе с WonderTales",
	DescEn:     "Read your favourite fairy tales together with Wonder Tales",
	Keywords:   "сказки, детские сказки, персонализированные сказки, WonderTales, создай свою собственную историю, сказки на ночь, рассказывание историй для детей, онлайн-сказки, дети как герои историй",
	KeywordsEn: "fairy tales, kids stories, personalized fairy tales, WonderTales, create your own story, bedtime stories, storytelling for children, online fairy tales, children as story heroes",
	Contrast:   true,
	CreatedAt:  &created,
}

func (p *PublicApi) Home(c echo.Context) error {
	tales, err := p.DB.FindStories(nil)

	if err != nil {
		log.Printf("Failed to load stories: %s\n", err.Error())
	}

	fmt.Println("utils.Htmx(c)", utils.Htmx(c), c.Request().Header.Get("HX-Request"), c.Request().Header.Get("hx-history-restore-request"))

	if utils.Htmx(c) {
		c.Response().Header().Set("HX-Retarget", "#search-result")

		return views.TalesSearchResult(tales).Render(c.Request().Context(), c.Response().Writer)
	}

	categories, err := p.DB.FindCategoriesOnMain(db.TALES_CATALOG)

	if err != nil {
		log.Printf("Failed to load categories: %s\n", err.Error())
	}

	props := views.TaleProps{
		BaseViewProps: staticHomePageProps,
		Stories:       tales,
		Categories:    categories,
	}

	c.Response().Header().Set("HX-Retarget", "body")

	return views.Tales(props).Render(c.Request().Context(), c.Response().Writer)
}

func (p *PublicApi) CategoryPage(c echo.Context) error {
	path := c.Param("path")

	if err := validatePath(path); err != nil {
		return c.String(http.StatusOK, "Not found :(")
		// return views.P404().Render(c.Request().Context(), c.Response().Writer)
	}

	category, err := p.DB.FindCategoryByPath(db.TALES_CATALOG, path)

	if err != nil {
		return c.String(http.StatusOK, "Not found :(")
		// return views.P404().Render(c.Request().Context(), c.Response().Writer)
	}

	props := views.CategoryPageProps{
		BaseViewProps: views.BaseViewProps{
			Title:      category.Name + " | WonderTales Blog",
			TitleEn:    category.NameEng + " | WonderTales Blog",
			Desc:       category.SeoDesc,
			DescEn:     category.SeoDescEng,
			Keywords:   category.SeoKeywords,
			KeywordsEn: category.SeoKeywordsEng,
			Image:      category.Cover,
			UpdatedAt:  &category.UpdatedAt,
			CreatedAt:  &category.CreatedAt,
		},
		Category: category,
	}

	return views.CategoryPage(props).Render(c.Request().Context(), c.Response().Writer)
}

func (p *PublicApi) StoryPage(c echo.Context) error {
	path := c.Param("path")

	if err := validatePath(path); err != nil {
		return c.String(http.StatusOK, "Not found :(")
		// return views.P404().Render(c.Request().Context(), c.Response().Writer)
	}

	story, err := p.DB.FindStoryByPath(path)

	if err != nil {
		return c.String(http.StatusOK, "Not found :(")
		// return views.P404().Render(c.Request().Context(), c.Response().Writer)
	}

	props := views.StoryPageProps{
		BaseViewProps: views.BaseViewProps{
			Title:      story.Name + " | WonderTales Blog",
			TitleEn:    story.NameEng + " | WonderTales Blog",
			Desc:       story.SeoDesc,
			DescEn:     story.SeoDescEng,
			Keywords:   story.SeoKeywords,
			KeywordsEn: story.SeoKeywordsEng,
			Image:      story.Cover,
			UpdatedAt:  &story.UpdatedAt,
			CreatedAt:  &story.CreatedAt,
		},
		Story: story,
	}

	return views.StoryPage(props).Render(c.Request().Context(), c.Response().Writer)
}

var staticArticlesPageProps = views.BaseViewProps{
	Title:      "Статьи для вас и ваших детей | WonderTales Blog",
	TitleEn:    "Articles for you and your children | WonderTales Blog",
	Desc:       "Изучите статьи о детском чтении и книгах на WonderTales. Откройте для себя советы, бесплатные детские книги онлайн и магию персонализированных историй, где ваш ребенок становится героем своей собственной сказки",
	DescEn:     "Explore articles about children's reading and books on WonderTales. Discover tips, free kids books online, and the magic of personalized stories where your child becomes the hero of their own fairy tale",
	Keywords:   "детские книги, детское чтение, бесплатные детские книги онлайн, персонализированные сказки на ночь, повествование с помощью искусственного интеллекта, WonderTales, сказки для детей, онлайн-книги с картинками, советы по чтению для детей, платформа для повествования",
	KeywordsEn: "children's books, kids reading, free kids books online, personalized bedtime stories, AI storytelling, WonderTales, fairy tales for children, online picture books, reading tips for kids, storytelling platform",
	Contrast:   true,
	CreatedAt:  &created,
}

func (p *PublicApi) ArticlesPage(c echo.Context) error {
	articles, err := p.DB.FindArticles(nil)

	if err != nil {
		log.Printf("Failed to load articles: %s\n", err.Error())
	}

	if utils.Htmx(c) {
		c.Response().Header().Set("HX-Retarget", "#search-result")

		return views.ArticlesSearchResult(articles).Render(c.Request().Context(), c.Response().Writer)
	}

	cats, err := p.DB.FindCategoriesOnMain(db.ARTICLES_CATALOG)

	if err != nil {
		log.Printf("Failed to load category for articles: %s", err.Error())
	}

	props := views.ArticlesPageProps{
		BaseViewProps: staticArticlesPageProps,
		Categories:    cats,
		Articles:      articles,
	}

	c.Response().Header().Set("HX-Retarget", "body")

	return views.ArticlesPage(props).Render(c.Request().Context(), c.Response().Writer)
}

func (p *PublicApi) ArticleCategoryPage(c echo.Context) error {
	path := c.Param("path")

	if err := validatePath(path); err != nil {
		return c.String(http.StatusOK, "Not found :(")
		// return views.P404().Render(c.Request().Context(), c.Response().Writer)
	}

	category, err := p.DB.FindCategoryByPath(db.ARTICLES_CATALOG, path)

	if err != nil {
		return c.String(http.StatusOK, "Not found :(")
		// return views.P404().Render(c.Request().Context(), c.Response().Writer)
	}

	props := views.ArticleCategoryPageProps{
		BaseViewProps: views.BaseViewProps{
			Title:      category.Name + " | WonderTales Blog",
			TitleEn:    category.NameEng + " | WonderTales Blog",
			Desc:       category.SeoDesc,
			DescEn:     category.SeoDescEng,
			Keywords:   category.SeoKeywords,
			KeywordsEn: category.SeoKeywordsEng,
			Image:      category.Cover,
			UpdatedAt:  &category.UpdatedAt,
			CreatedAt:  &category.CreatedAt,
		},
		Category: category,
	}

	return views.ArticleCategoryPage(props).Render(c.Request().Context(), c.Response().Writer)
}

func (p *PublicApi) ArticlePage(c echo.Context) error {
	path := c.Param("path")

	if err := validatePath(path); err != nil {
		return c.String(http.StatusOK, "Not found :(")
		// return views.P404().Render(c.Request().Context(), c.Response().Writer)
	}

	article, err := p.DB.FindArticleByPath(path)

	if err != nil {
		return c.String(http.StatusOK, "Not found :(")
		// return views.P404().Render(c.Request().Context(), c.Response().Writer)
	}

	props := views.ArticlePageProps{
		BaseViewProps: views.BaseViewProps{
			Title:      article.Name + " | WonderTales Blog",
			TitleEn:    article.NameEng + " | WonderTales Blog",
			Desc:       article.SeoDesc,
			DescEn:     article.SeoDescEng,
			Keywords:   article.SeoKeywords,
			KeywordsEn: article.SeoKeywordsEng,
			Image:      article.Cover,
			UpdatedAt:  &article.UpdatedAt,
			CreatedAt:  &article.CreatedAt,
		},
		Article: article,
	}

	return views.ArticlePage(props).Render(c.Request().Context(), c.Response().Writer)
}
