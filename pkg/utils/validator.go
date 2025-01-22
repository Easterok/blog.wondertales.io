package utils

import (
	"strings"

	"github.com/easterok/blogs/pkg/db"
)

func ValidateRuTale(tale *db.Story) ([]string, []string) {
	hasTitleRu := tale.Name != "" && !strings.HasPrefix(tale.Name, "Название")
	hasPrefixRu := tale.Prefix != "" && tale.Prefix != "Какой-то текст"
	hasPostfixRu := tale.Postfix != "" && tale.Postfix != "Какой-то текст"

	hasSeoDescr := tale.SeoDesc != ""
	hasSeoKey := tale.SeoKeywords != ""

	acc := []string{}
	seo := []string{}

	if !hasTitleRu {
		acc = append(acc, "Пропущено название")
	}
	if !hasPrefixRu {
		acc = append(acc, "Не указан текст до промо раздела")
	} else if len(tale.Prefix) < 20 {
		acc = append(acc, "Текста до промо слишком мало")
	}

	if !hasPostfixRu {
		acc = append(acc, "Не указан текст после промо раздела")
	} else if len(tale.Postfix) < 20 {
		acc = append(acc, "Текста после промо слишком мало")
	}

	if !hasSeoDescr {
		seo = append(seo, "Пропущено описание")
	} else if len(tale.SeoDesc) < 10 {
		seo = append(seo, "Описание слишком короткое")
	}

	if !hasSeoKey {
		seo = append(seo, "Пропущены ключевые слова")
	} else if len(tale.SeoKeywords) < 10 {
		seo = append(seo, "Текст для ключевых слов слишком короткий")
	}

	return acc, seo
}

func ValidateEnTale(tale *db.Story) ([]string, []string) {
	hasTitleEn := tale.NameEng != "" && !strings.HasPrefix(tale.NameEng, "Title")
	hasPrefixEn := tale.PrefixEng != "" && tale.PrefixEng != "Some text"
	hasPostfixEn := tale.PostfixEng != "" && tale.PostfixEng != "Some text"

	hasSeoDescrEn := tale.SeoDescEng != ""
	hasSeoKeyEn := tale.SeoKeywordsEng != ""

	acc := []string{}
	seo := []string{}

	if !hasTitleEn {
		acc = append(acc, "Пропущено название")
	}

	if !hasPrefixEn {
		acc = append(acc, "Не указан текст до промо раздела")
	} else if len(tale.PrefixEng) < 20 {
		acc = append(acc, "Текста до промо слишком мало")
	}

	if !hasPostfixEn {
		acc = append(acc, "Не указан текст после промо раздела")
	} else if len(tale.PostfixEng) < 20 {
		acc = append(acc, "Текста после промо слишком мало")
	}

	if !hasSeoDescrEn {
		seo = append(seo, "Пропущено описание")
	} else if len(tale.SeoDescEng) < 10 {
		seo = append(seo, "Описание слишком короткое")
	}

	if !hasSeoKeyEn {
		seo = append(seo, "Пропущены ключевые слова")
	} else if len(tale.SeoKeywordsEng) < 10 {
		seo = append(seo, "Текст для ключевых слов слишком короткий")
	}

	return acc, seo
}

func ValidateRuCatalog(tale *db.Catalog) ([]string, []string) {
	hasTitleRu := tale.Name != "" && !strings.HasPrefix(tale.Name, "Название")

	hasSeoDescr := tale.SeoDesc != ""
	hasSeoKey := tale.SeoKeywords != ""

	acc := []string{}
	seo := []string{}

	if !hasTitleRu {
		acc = append(acc, "Пропущено название")
	}

	if !hasSeoDescr {
		seo = append(seo, "Пропущено описание")
	} else if len(tale.SeoDesc) < 10 {
		seo = append(seo, "Описание слишком короткое")
	}

	if !hasSeoKey {
		seo = append(seo, "Пропущены ключевые слова")
	} else if len(tale.SeoKeywords) < 10 {
		seo = append(seo, "Описание слишком короткое")
	}

	return acc, seo
}

func ValidateEnCatalog(tale *db.Catalog) ([]string, []string) {
	hasTitleEn := tale.NameEng != "" && !strings.HasPrefix(tale.NameEng, "Title")

	hasSeoDescrEn := tale.SeoDescEng != ""
	hasSeoKeyEn := tale.SeoKeywordsEng != ""

	acc := []string{}
	seo := []string{}

	if !hasTitleEn {
		acc = append(acc, "Пропущено название")
	}

	if !hasSeoDescrEn {
		seo = append(seo, "Пропущено описание")
	} else if len(tale.SeoDescEng) < 10 {
		seo = append(seo, "Описание слишком короткое")
	}

	if !hasSeoKeyEn {
		seo = append(seo, "Пропущены ключевые слова")
	} else if len(tale.SeoKeywordsEng) < 10 {
		seo = append(seo, "Текст для ключевых слов слишком короткий")
	}

	return acc, seo
}

func ValidateRuArticle(tale *db.Article) ([]string, []string) {
	hasTitleRu := tale.Name != "" && !strings.HasPrefix(tale.Name, "Название")
	hasPrefixRu := tale.Prefix != "" && tale.Prefix != "Какой-то текст"
	hasPostfixRu := tale.Postfix != "" && tale.Postfix != "Какой-то текст"

	hasSeoDescr := tale.SeoDesc != ""
	hasSeoKey := tale.SeoKeywords != ""

	acc := []string{}
	seo := []string{}

	if !hasTitleRu {
		acc = append(acc, "Пропущено название")
	}
	if !hasPrefixRu {
		acc = append(acc, "Не указан текст до оглавления")
	} else if len(tale.Prefix) < 20 {
		acc = append(acc, "Текста до оглавления слишком мало")
	}

	if !hasPostfixRu {
		acc = append(acc, "Не указан текст после оглавления")
	} else if len(tale.Postfix) < 20 {
		acc = append(acc, "Текста после оглавления слишком мало")
	}

	if !hasSeoDescr {
		seo = append(seo, "Пропущено описание")
	} else if len(tale.SeoDesc) < 10 {
		seo = append(seo, "Описание слишком короткое")
	}

	if !hasSeoKey {
		seo = append(seo, "Пропущены ключевые слова")
	} else if len(tale.SeoKeywords) < 10 {
		seo = append(seo, "Текст для ключевых слов слишком короткий")
	}

	return acc, seo
}

func ValidateEnArticle(tale *db.Article) ([]string, []string) {
	hasTitleEn := tale.NameEng != "" && !strings.HasPrefix(tale.NameEng, "Title")
	hasPrefixEn := tale.PrefixEng != "" && tale.PrefixEng != "Some text"
	hasPostfixEn := tale.PostfixEng != "" && tale.PostfixEng != "Some text"

	hasSeoDescrEn := tale.SeoDescEng != ""
	hasSeoKeyEn := tale.SeoKeywordsEng != ""

	acc := []string{}
	seo := []string{}

	if !hasTitleEn {
		acc = append(acc, "Пропущено название")
	}

	if !hasPrefixEn {
		acc = append(acc, "Не указан текст до оглавления")
	} else if len(tale.PrefixEng) < 20 {
		acc = append(acc, "Текста до оглавления слишком мало")
	}

	if !hasPostfixEn {
		acc = append(acc, "Не указан текст после оглавления")
	} else if len(tale.PostfixEng) < 20 {
		acc = append(acc, "Текста после оглавления слишком мало")
	}

	if !hasSeoDescrEn {
		seo = append(seo, "Пропущено описание")
	} else if len(tale.SeoDescEng) < 10 {
		seo = append(seo, "Описание слишком короткое")
	}

	if !hasSeoKeyEn {
		seo = append(seo, "Пропущены ключевые слова")
	} else if len(tale.SeoKeywordsEng) < 10 {
		seo = append(seo, "Текст для ключевых слов слишком короткий")
	}

	return acc, seo
}
