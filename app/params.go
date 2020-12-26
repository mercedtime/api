package app

import (
	"fmt"

	"github.com/doug-martin/goqu/v9"
	"github.com/gin-gonic/gin"
)

// PageParams are url params for api pagination
type PageParams struct {
	Limit  uint `form:"limit"  query:"limit"  db:"limit"`
	Offset uint `form:"offset" query:"offset" db:"offset"`
}

func (pp *PageParams) toExpr() goqu.Ex {
	ex := goqu.Ex{}
	if pp.Limit != 0 {
		ex["limit"] = pp.Limit
	}
	if pp.Offset != 0 {
		ex["offset"] = pp.Offset
	}
	return ex
}

func (pp *PageParams) appendSelect(stmt *goqu.SelectDataset) *goqu.SelectDataset {
	if pp.Limit != 0 {
		stmt = stmt.Limit(pp.Limit)
	}
	if pp.Offset != 0 {
		stmt = stmt.Offset(pp.Offset)
	}
	return stmt
}

func (pp *PageParams) asSQL(stmtIndex int) (string, []interface{}, int) {
	var (
		q    string
		args = make([]interface{}, 0, 2)
	)
	if pp.Limit != 0 {
		args = append(args, pp.Limit)
		q += fmt.Sprintf(" LIMIT $%d", stmtIndex)
		stmtIndex++
	}
	if pp.Offset != 0 {
		args = append(args, pp.Offset)
		q += fmt.Sprintf(" OFFSET $%d", stmtIndex)
		stmtIndex++
	}
	return q, args, stmtIndex
}

// Expression implements the goqu.Expression interface
func (pp *PageParams) Expression() goqu.Expression { return pp.toExpr() }

// Clone implements the goqu.Expression interface
func (pp *PageParams) Clone() goqu.Expression { return pp.toExpr() }

// SemesterParams is a structure that defines
// parameters that control which courses are returned from a query
type SemesterParams struct {
	Year    int    `form:"year" uri:"year" query:"year" db:"year"`
	Term    string `form:"term" uri:"term" query:"term" db:"term_id"`
	Subject string `form:"subject" query:"subject" db:"subject"`
}

func (sp *SemesterParams) toExpr() goqu.Ex {
	ex := goqu.Ex{}
	if sp.Year != 0 {
		ex["year"] = sp.Year
	}
	if sp.Term != "" {
		if id := getTermID(sp.Term); id != 0 {
			ex["term_id"] = id
		}
	}
	if sp.Subject != "" {
		ex["subject"] = sp.Subject
	}
	return ex
}

// Expression implements the goqu.Expression interface
func (sp *SemesterParams) Expression() goqu.Expression { return sp.toExpr() }

// Clone implements the goqu.Expression interface
func (sp *SemesterParams) Clone() goqu.Expression { return sp.toExpr() }

func (sp *SemesterParams) bind(c *gin.Context) (err error) {
	if err = c.BindQuery(sp); err != nil {
		return err
	}
	if err = c.BindUri(sp); err != nil {
		return err
	}
	return nil
}
