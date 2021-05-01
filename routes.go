package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	srchx "github.com/abasse/libsrchx"

	"github.com/blevesearch/bleve"
	"github.com/blevesearch/bleve/search/query"
	"github.com/labstack/echo"
)

/**
 * deleteDoc - delete the json file from filesystem
 */
func deleteDoc(ndx string, typ string, id string) {

	thepath := Jsonpath + ndx + "/" + typ + "/" + id
	files, err := filepath.Glob(thepath + "/*")
	if err != nil {
		panic(err)
	}
	for _, f := range files {
		if err := os.Remove(f); err != nil {
			panic(err)
		}
	}

	err = os.Remove(thepath)
	if err != nil {
		panic(err)
	}

	return
}

/**
 * saveDoc - save the json to the filesystem
 */
func saveDoc(doc map[string]interface{}, ndx string, typ string, id string) {

	thepath := Jsonpath + ndx + "/" + typ + "/" + id

	if _, err := os.Stat(thepath); os.IsNotExist(err) {
		os.MkdirAll(thepath, 0777)
	}

	f, err := os.Create(thepath + "/" + id + ".json")
	if err != nil {
		panic(err)
	}

	jsonString, err := json.Marshal(doc)
	if err != nil {
		fmt.Println(err)
	}

	_, err = f.Write(jsonString)
	if err != nil {
		fmt.Println(err)
	}

	return
}

/**
 * routeHome - the home route
 */
func routeHome(c echo.Context) error {
	return c.JSON(200, map[string]interface{}{
		"success": true,
		"message": "I'm ready",
	})
}

/**
 * routeIndex - index a document
 */
func routeIndex(c echo.Context) error {
	var doc map[string]interface{}
	if err := c.Bind(&doc); err != nil {
		return c.JSON(400, map[string]interface{}{
			"success": false,
			"error":   err.Error(),
		})
	}

	ndx, typ, id := c.Param("index"), c.Param("type"), c.Param("id")
	index, err := store.GetIndex(ndx + "/" + typ)
	if err != nil {
		return c.JSON(500, map[string]interface{}{
			"success": false,
			"error":   err.Error(),
		})
	}

	if strings.ToLower(id) == "new" {
		doc["id"] = ""
	} else {
		doc["id"] = id
	}

	doc, err = index.Put(doc)
	if err != nil {
		return c.JSON(500, map[string]interface{}{
			"success": false,
			"error":   err.Error(),
		})
	}

	if StoreJson {
		saveDoc(doc, ndx, typ, doc["id"].(string))
	}

	return c.JSON(200, map[string]interface{}{
		"success": true,
		"payload": doc,
	})

}

/**
 * routeBatchIndex - index multiple documents
 */
func routeBatchIndex(c echo.Context) error {
	var docs []map[string]interface{}
	if err := c.Bind(&docs); err != nil {
		return c.JSON(400, map[string]interface{}{
			"success": false,
			"error":   err.Error(),
		})
	}

	ndx, typ := c.Param("index"), c.Param("type")
	index, err := store.GetIndex(ndx + "/" + typ)
	if err != nil {
		return c.JSON(500, map[string]interface{}{
			"success": false,
			"error":   err.Error(),
		})
	}

	success := []string{}
	errs := []string{}

	for _, doc := range docs {
		doc, err = index.Put(doc)
		if err == nil {
			success = append(success, doc["id"].(string))
		} else {
			errs = append(errs, err.Error())
		}
	}

	return c.JSON(200, map[string]interface{}{
		"success": len(success) > len(errs),
		"errors":  errs,
		"payload": success,
	})
}

/**
 * routeGet - get a document
 */
func routeGet(c echo.Context) error {
	ndx, typ, id := c.Param("index"), c.Param("type"), c.Param("id")
	index, err := store.GetIndex(ndx + "/" + typ)
	if err != nil {
		return c.JSON(404, map[string]interface{}{
			"success": false,
			"error":   err.Error(),
		})
	}

	doc, err := index.Get(id)
	if err != nil {
		return c.JSON(404, map[string]interface{}{
			"success": false,
			"error":   err.Error(),
		})
	}

	return c.JSON(200, map[string]interface{}{
		"success": true,
		"payload": doc,
	})
}

/**
 * routeDelete - delete a document
 */
func routeDelete(c echo.Context) error {
	ndx, typ, id := c.Param("index"), c.Param("type"), c.Param("id")
	index, err := store.GetIndex(ndx + "/" + typ)
	if err != nil {
		return c.JSON(404, map[string]interface{}{
			"success": false,
			"error":   err.Error(),
		})
	}

	index.Delete(id)

	if StoreJson {
		deleteDoc(ndx, typ, id)
	}

	return c.JSON(200, map[string]interface{}{
		"success": true,
	})
}

/**
 * routeSearch - search for documents
 */
func routeSearch(c echo.Context) error {
	var input struct {
		QueryString string `json:"query"`

		srchx.Query

		Join []struct {
			From string `json:"from"`

			*srchx.Join
		} `json:"join"`
	}

	var q query.Query

	if err := c.Bind(&input); err != nil {
		return c.JSON(400, map[string]interface{}{
			"success": false,
			"error":   err.Error(),
		})
	}

	ndx, typ := c.Param("index"), c.Param("type")
	index, err := store.GetIndex(ndx + "/" + typ)
	if err != nil {
		return c.JSON(404, map[string]interface{}{
			"success": false,
			"error":   err.Error(),
		})
	}

	if input.QueryString != "" {
		q = query.Query(bleve.NewQueryStringQuery(input.QueryString))
	}

	if strings.Contains(input.QueryString, "*") {
		q = query.Query(bleve.NewWildcardQuery("*" + strings.Trim(strings.ToLower(input.QueryString), "*") + "*"))
	}

	if q == nil {
		q = query.Query(bleve.NewMatchAllQuery())
	}

	joins := []*srchx.Join{}

	for _, join := range input.Join {
		if join.From != "" {
			ndx, e := store.GetIndex(join.From)
			if e != nil {

				return c.JSON(404, map[string]interface{}{
					"success": false,
					"error":   e.Error(),
				})
			}
			join.Join.Src = ndx
			joins = append(joins, join.Join)
		}
	}

	req := &srchx.Query{
		Query:  q,
		Offset: input.Offset,
		Size:   input.Size,
		Sort:   input.Sort,
		Join:   joins,
	}

	res, err := index.Search(req)

	if err != nil {
		return c.JSON(500, map[string]interface{}{
			"success": false,
			"error":   err.Error(),
		})
	}

	return c.JSON(200, map[string]interface{}{
		"success": true,
		"payload": res,
	})
}

func routeAggregate(c echo.Context) error {
	var input struct {
		QueryString string `json:"query"`

		srchx.Query
	}

	var q query.Query

	if err := c.Bind(&input); err != nil {
		return c.JSON(400, map[string]interface{}{
			"success": false,
			"error":   err.Error(),
		})
	}

	ndx, typ := c.Param("index"), c.Param("type")
	index, err := store.GetIndex(ndx + "/" + typ)
	if err != nil {
		return c.JSON(404, map[string]interface{}{
			"success": false,
			"error":   err.Error(),
		})
	}

	if input.QueryString != "" {
		q = query.Query(bleve.NewQueryStringQuery(input.QueryString))
	}

	if q == nil {
		q = query.Query(bleve.NewMatchAllQuery())
	}

	return c.JSON(200, map[string]interface{}{
		"success": true,
		"payload": index.Aggregate(&srchx.Query{Query: q}, c.Param("field"), c.Param("func")),
	})
}
