package db

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"golang.org/x/text/unicode/norm"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type DB struct {
	g *gorm.DB
}

type CatalogType = string

const (
	TALES_CATALOG    CatalogType = "tale"
	ARTICLES_CATALOG CatalogType = "article"

	LIMIT_CATEGORY_ON_MAIN = 10
)

var (
	TRUE  = ToCheckboxValue("on")
	FALSE = ToCheckboxValue("off")
)

type Article struct {
	ID uint `gorm:"primarykey"`

	CreatedAt time.Time
	UpdatedAt time.Time

	Path    string `gorm:"uniqueIndex"`
	PathEng string `gorm:"uniqueIndex"`

	Cover string

	// Content of story
	Prefix     string
	PrefixEng  string
	Postfix    string
	PostfixEng string

	SeoDesc        string
	SeoDescEng     string
	SeoKeywords    string
	SeoKeywordsEng string

	Name    string
	NameEng string

	Published *bool

	Catalogs []CatalogArticles `gorm:"foreignKey:ArticleID;constraint:onDelete:CASCADE"`
}

type Story struct {
	ID uint `gorm:"primarykey"`

	CreatedAt time.Time
	UpdatedAt time.Time

	Path    string `gorm:"uniqueIndex"`
	PathEng string `gorm:"uniqueIndex"`

	Cover string

	// Content of story
	Prefix     string
	PrefixEng  string
	Postfix    string
	PostfixEng string

	SeoDesc        string
	SeoDescEng     string
	SeoKeywords    string
	SeoKeywordsEng string

	Name    string
	NameEng string

	Published *bool

	Catalogs []CatalogStories `gorm:"foreignKey:StoryID;constraint:onDelete:CASCADE"`
}

type Catalog struct {
	ID uint `gorm:"primarykey"`

	CreatedAt time.Time
	UpdatedAt time.Time

	Cover string

	Name    string
	Type    CatalogType
	NameEng string

	SeoDesc        string
	SeoDescEng     string
	SeoKeywords    string
	SeoKeywordsEng string

	Path    string `gorm:"uniqueIndex"`
	PathEng string `gorm:"uniqueIndex"`

	Hidden     *bool `gorm:"default:true"`
	ShowOnMain *bool `gorm:"index:idx_show_on_main"`

	CatalogsStories  []CatalogStories  `gorm:"foreignKey:CatalogID;constraint:onDelete:CASCADE"`
	CatalogsArticles []CatalogArticles `gorm:"foreignKey:CatalogID;constraint:onDelete:CASCADE"`
}

type CatalogStories struct {
	ID uint `gorm:"primarykey"`

	CatalogID uint    `gorm:"not null;index:idx_category_story,unique"`
	Catalog   Catalog `gorm:"foreignKey:CatalogID"`

	StoryID uint  `gorm:"not null;index:idx_category_story,unique"`
	Story   Story `gorm:"foreignKey:StoryID"`
}

type CatalogArticles struct {
	ID uint `gorm:"primarykey"`

	CatalogID uint    `gorm:"not null;index:idx_category_article,unique"`
	Catalog   Catalog `gorm:"foreignKey:CatalogID"`

	ArticleID uint    `gorm:"not null;index:idx_category_article,unique"`
	Article   Article `gorm:"foreignKey:ArticleID"`
}

func Connect(dns string, config *gorm.Config) (*DB, error) {
	g, err := gorm.Open(postgres.Open(dns), config)

	if err != nil {
		return nil, err
	}

	err = g.AutoMigrate(&Catalog{}, &Story{}, &Article{}, &CatalogStories{}, &CatalogArticles{})

	if err != nil {
		return nil, err
	}

	return &DB{g: g}, nil
}

func (db *DB) CreateDraftStory(categoryId string) (*Story, error) {
	now := time.Now()

	var cat Catalog

	if categoryId != "" {
		id, err := strconv.ParseUint(categoryId, 10, 32)

		if err != nil {
			return nil, err
		}

		if result := db.g.Where(&Catalog{ID: uint(id)}).First(&cat); result.Error != nil {
			return nil, result.Error
		}
	}

	draft := Story{
		Published: FALSE,
		CreatedAt: now,
		UpdatedAt: now,

		Prefix:     "Какой-то текст...",
		PrefixEng:  "Some text...",
		Postfix:    "Какой-то текст...",
		PostfixEng: "Some text...",
		Name:       fmt.Sprintf("Название %d", now.Unix()),
		NameEng:    fmt.Sprintf("Title %d", now.Unix()),
	}

	draft.Path = NameToPath(draft.Name)
	draft.PathEng = NameToPath(draft.NameEng)

	if result := db.g.Create(&draft); result.Error != nil {
		return nil, result.Error
	}

	if cat.ID != 0 {
		if _, err := db.CreateStoryConnection(draft.ID, cat.ID); err != nil {
			return nil, err
		}
	}

	return &draft, nil
}

func (db *DB) CreateDraftArticle(categoryId string) (*Article, error) {
	now := time.Now()

	var cat Catalog

	if categoryId != "" {
		id, err := strconv.ParseUint(categoryId, 10, 32)

		if err != nil {
			return nil, err
		}

		if result := db.g.Where(&Catalog{ID: uint(id)}).First(&cat); result.Error != nil {
			return nil, result.Error
		}
	}

	draft := Article{
		Published: FALSE,
		CreatedAt: now,
		UpdatedAt: now,

		Prefix:     "Какой-то текст...",
		PrefixEng:  "Some text...",
		Postfix:    "Какой-то текст...",
		PostfixEng: "Some text...",
		Name:       fmt.Sprintf("Название %d", now.Unix()),
		NameEng:    fmt.Sprintf("Title %d", now.Unix()),
	}

	draft.Path = NameToPath(draft.Name)
	draft.PathEng = NameToPath(draft.NameEng)

	if result := db.g.Create(&draft); result.Error != nil {
		return nil, result.Error
	}

	if cat.ID != 0 {
		if _, err := db.CreateArticleConnection(draft.ID, cat.ID); err != nil {
			return nil, err
		}
	}

	return &draft, nil
}

func (db *DB) GetStoryById(id uint) (*Story, error) {
	var story Story

	if result := db.g.Where(&Story{ID: id}).First(&story); result.Error != nil {
		return nil, result.Error
	}

	if err := db.PostloadCatalogsStories(&story, false); err != nil {
		return nil, err
	}

	return &story, nil
}

func (db *DB) GetArticleById(id uint) (*Article, error) {
	var article Article

	if result := db.g.Where(&Article{ID: id}).First(&article); result.Error != nil {
		return nil, result.Error
	}

	if err := db.PostloadCatalogsArticles(&article, false); err != nil {
		return nil, err
	}

	return &article, nil
}

func (db *DB) UpdateTale(id uint, c *Story) error {
	now := time.Now()

	c.UpdatedAt = now

	if c.Name != "" {
		c.Path = NameToPath(c.Name)
	}

	if c.NameEng != "" {
		c.PathEng = NameToPath(c.NameEng)
	}

	return db.g.Model(&Story{}).Where(&Story{ID: id}).Updates(c).Error
}

func (db *DB) UpdateArticle(id uint, c *Article) error {
	now := time.Now()

	c.UpdatedAt = now

	if c.Name != "" {
		c.Path = NameToPath(c.Name)
	}

	if c.NameEng != "" {
		c.PathEng = NameToPath(c.NameEng)
	}

	return db.g.Model(&Article{}).
		Where(&Article{ID: id}).
		Updates(c).
		Error
}

func (db *DB) DeleteStory(id uint) error {
	return db.g.Delete(&Story{ID: id}).Error
}

func (db *DB) DeleteArticle(id uint) error {
	return db.g.Delete(&Article{ID: id}).Error
}

func (db *DB) DeleteStoryConnection(id uint) error {
	return db.g.Delete(&CatalogStories{ID: id}).Error
}

func (db *DB) UpdateStoryConnection(id, newCatalogId uint) error {
	return db.g.Model(&CatalogStories{}).
		Where(&CatalogStories{ID: id}).
		Updates(&CatalogStories{CatalogID: newCatalogId}).
		Error
}

func (db *DB) FindCatalogStoriesByStoryId(storyId uint) (*[]CatalogStories, error) {
	var rows []CatalogStories
	if err := db.g.Model(&CatalogStories{}).
		Where(&CatalogStories{StoryID: storyId}).
		Find(&rows).
		Error; err != nil {
		return nil, err
	}
	return &rows, nil
}

func (db *DB) DeleteArticleConnection(id uint) error {
	return db.g.Delete(&CatalogArticles{ID: id}).Error
}

func (db *DB) UpdateArticleConnection(id, newCatalogId uint) error {
	return db.g.Model(&CatalogArticles{}).
		Where(&CatalogArticles{ID: id}).
		Updates(&CatalogArticles{CatalogID: newCatalogId}).
		Error
}

func (db *DB) FindCatalogArticlesByArticleId(articleId uint) (*[]CatalogArticles, error) {
	var rows []CatalogArticles
	if err := db.g.Model(&CatalogArticles{}).
		Where(&CatalogArticles{ArticleID: articleId}).
		Find(&rows).
		Error; err != nil {
		return nil, err
	}
	return &rows, nil
}

func (db *DB) CreateStoryConnection(storyId, catalogId uint) (*CatalogStories, error) {
	s := CatalogStories{CatalogID: catalogId}

	if err := db.g.Model(&Story{ID: storyId}).
		Association("Catalogs").
		Append(&s); err != nil {
		return nil, err
	}

	return &s, nil
}

func (db *DB) CreateArticleConnection(articleId, catalogId uint) (*CatalogArticles, error) {
	s := CatalogArticles{CatalogID: catalogId}

	if err := db.g.Model(&Article{ID: articleId}).
		Association("Catalogs").
		Append(&s); err != nil {
		return nil, err
	}

	return &s, nil
}

func (db *DB) GetAllCatalogItems(t CatalogType) *[]Catalog {
	var rows []Catalog

	db.g.Where(Catalog{Type: t}).Order("updated_at desc").Find(&rows)

	return &rows
}

func (db *DB) CreateCatalog(cat *Catalog) error {
	now := time.Now()

	u := fmt.Sprintf("%d", now.Unix())

	cat.CreatedAt = now
	cat.UpdatedAt = now

	cat.Name = strings.TrimSpace(cat.Name) + u
	cat.NameEng = strings.TrimSpace(cat.NameEng) + u
	cat.Path = NameToPath(cat.Name)
	cat.PathEng = NameToPath(cat.NameEng)

	result := db.g.Create(&cat)

	if result.Error != nil {
		return result.Error
	}

	return nil
}

func (db *DB) UpdateCatalog(id uint, cat *Catalog) error {
	now := time.Now()

	cat.UpdatedAt = now

	if cat.Name != "" {
		cat.Path = NameToPath(cat.Name)
	}

	if cat.NameEng != "" {
		cat.PathEng = NameToPath(cat.NameEng)
	}

	res := db.g.Model(&Catalog{}).Where(&Catalog{ID: id}).Updates(cat)

	if res.Error != nil {
		return res.Error
	}

	return nil
}

func (db *DB) FindCategoryByPath(t CatalogType, path string) (*Catalog, error) {
	var cat Catalog

	result := db.g.Where("hidden = false and (path = ? or path_eng = ?)", path, path).First(&cat)

	if result.Error != nil {
		return nil, result.Error
	}

	if t == TALES_CATALOG {
		if err := db.g.
			InnerJoins("Story", db.g.Where(&Story{Published: TRUE}).Model(&Story{})).
			Where(&CatalogStories{CatalogID: cat.ID}).
			Order("created_at desc").
			Find(&cat.CatalogsStories).Error; err != nil {
			return nil, err
		}
	} else if t == ARTICLES_CATALOG {
		if err := db.g.
			InnerJoins("Article", db.g.Where(&Article{Published: TRUE}).Model(&Article{})).
			Where(&CatalogArticles{CatalogID: cat.ID}).
			Order("created_at desc").
			Find(&cat.CatalogsArticles).Error; err != nil {
			return nil, err
		}
	}

	return &cat, nil
}

func (db *DB) FindCategoryById(id uint) (*Catalog, error) {
	var cat Catalog

	result := db.g.First(&cat, id)

	if result.Error != nil {
		return nil, result.Error
	}

	if cat.Type == TALES_CATALOG {
		if err := db.g.
			Joins("Story").
			Where(&CatalogStories{CatalogID: cat.ID}).
			Order("created_at desc").
			Find(&cat.CatalogsStories).Error; err != nil {
			return nil, err
		}
	} else if cat.Type == ARTICLES_CATALOG {
		if err := db.g.
			Joins("Article").
			Where(&CatalogArticles{CatalogID: cat.ID}).
			Order("created_at desc").
			Find(&cat.CatalogsArticles).Error; err != nil {
			return nil, err
		}
	}

	return &cat, nil
}

func (db *DB) DeleteCategory(id uint) error {
	return db.g.Select(clause.Associations).Delete(&Catalog{ID: id}).Error
}

func (db *DB) FindStories(q *string) (*[]Story, error) {
	var rows []Story

	if result := db.g.Model(&Story{}).
		Where(&Story{Published: TRUE}).
		Order("created_at desc").
		Find(&rows); result.Error != nil {
		return nil, result.Error
	}

	return &rows, nil
}

func (db *DB) FindAdminStories() (*[]Story, error) {
	var rows []Story

	result := db.g.Model(&Story{}).
		Preload("Catalogs").
		Preload("Catalogs.Catalog").
		Order("created_at desc").
		Find(&rows)

	if result.Error != nil {
		return nil, result.Error
	}

	return &rows, nil
}

func (db *DB) FindArticles(q *string) (*[]Article, error) {
	var rows []Article

	if result := db.g.Model(&Article{}).
		Where(&Article{Published: TRUE}).
		Order("created_at desc").
		Find(&rows); result.Error != nil {
		return nil, result.Error
	}

	return &rows, nil
}

func (db *DB) FindAdminArticles() (*[]Article, error) {
	var rows []Article

	result := db.g.Model(&Article{}).
		Preload("Catalogs").
		Preload("Catalogs.Catalog").
		Order("created_at desc").
		Find(&rows)

	if result.Error != nil {
		return nil, result.Error
	}

	return &rows, nil
}

func (db *DB) FindCategoriesOnMain(t CatalogType) (*[]Catalog, error) {
	var rows []Catalog

	result := db.g.Model(&Catalog{}).
		Where(&Catalog{Hidden: FALSE, Type: t, ShowOnMain: TRUE}).
		Order("created_at desc").
		Limit(LIMIT_CATEGORY_ON_MAIN).
		Find(&rows)

	if result.Error != nil {
		return nil, result.Error
	}

	return &rows, nil
}

func (db *DB) FindNotHiddenCategories(t CatalogType) (*[]Catalog, error) {
	var rows []Catalog

	result := db.g.Model(&Catalog{}).
		Where(&Catalog{Hidden: FALSE, Type: t}).
		Order("created_at desc").
		Find(&rows)

	if result.Error != nil {
		return nil, result.Error
	}

	return &rows, nil
}

func (db *DB) PostloadCatalogsStories(story *Story, onlyShownQuery bool) error {
	q := db.g.Model(&Catalog{})

	if onlyShownQuery {
		q = db.g.Where(&Catalog{Hidden: FALSE}).Model(&Catalog{})
	}

	return db.g.
		InnerJoins("Catalog", q).
		Where(&CatalogStories{StoryID: story.ID}).
		Order("created_at desc").
		Find(&story.Catalogs).Error
}

func (db *DB) PostloadCatalogsArticles(article *Article, onlyShownQuery bool) error {
	q := db.g.Model(&Catalog{})

	if onlyShownQuery {
		q = db.g.Where(&Catalog{Hidden: FALSE}).Model(&Catalog{})
	}

	return db.g.
		InnerJoins("Catalog", q).
		Where(&CatalogArticles{ArticleID: article.ID}).
		Order("created_at desc").
		Find(&article.Catalogs).Error
}

func (db *DB) FindStoryByPath(path string) (*Story, error) {
	var story Story

	if result := db.g.
		Model(&Story{}).
		Where("published = true and (path = ? or path_eng = ?)", path, path).
		First(&story); result.Error != nil {
		return nil, result.Error
	}

	if err := db.PostloadCatalogsStories(&story, true); err != nil {
		return nil, err
	}

	return &story, nil
}

func (db *DB) FindArticleByPath(path string) (*Article, error) {
	var article Article

	if result := db.g.
		Model(&Article{}).
		Where("published = true and (path = ? or path_eng = ?)", path, path).
		First(&article); result.Error != nil {
		return nil, result.Error
	}

	if err := db.PostloadCatalogsArticles(&article, true); err != nil {
		return nil, err
	}

	return &article, nil
}

func ToCheckboxValue(s string) *bool {
	var res *bool

	if s != "" {
		r := s == "on"

		res = &r
	}

	return res
}

var regexpPath, _ = regexp.Compile("[^a-zа-я0-9]+")

var ruToEng = map[rune]string{
	'а': "a", 'б': "b", 'в': "v", 'г': "g", 'д': "d",
	'е': "e", 'ё': "yo", 'ж': "zh", 'з': "z", 'и': "i",
	'й': "y", 'к': "k", 'л': "l", 'м': "m", 'н': "n",
	'о': "o", 'п': "p", 'р': "r", 'с': "s", 'т': "t",
	'у': "u", 'ф': "f", 'х': "kh", 'ц': "ts", 'ч': "ch",
	'ш': "sh", 'щ': "shch", 'ъ': "", 'ы': "y", 'ь': "",
	'э': "e", 'ю': "yu", 'я': "ya",
}

func Translate(s string) string {
	var builder strings.Builder
	for _, char := range s {
		if english, found := ruToEng[char]; found {
			builder.WriteString(english)
		} else {
			builder.WriteRune(char)
		}
	}
	return builder.String()
}

func NameToPath(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	s = strings.ReplaceAll(norm.NFC.String(s), "ё", "е")
	sanitized := regexpPath.ReplaceAllString(s, "-")

	return Translate(strings.Trim(sanitized, "-"))
}
