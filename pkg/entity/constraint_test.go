package entity

import (
	"testing"

	"github.com/openflagr/flagr/swagger_gen/models"
	"github.com/stretchr/testify/assert"
)

func TestConstraintToExpr(t *testing.T) {
	t.Run("empty case", func(t *testing.T) {
		c := Constraint{}
		expr, err := c.ToExpr()
		assert.Error(t, err)
		assert.Nil(t, expr)
	})

	t.Run("not supported operator case", func(t *testing.T) {
		c := Constraint{
			SegmentID: 0,
			Property:  "dl_state",
			Operator:  "===",
			Value:     "\"CA\"",
		}
		expr, err := c.ToExpr()
		assert.Error(t, err)
		assert.Nil(t, expr)
	})

	t.Run("parse error - invalid ]", func(t *testing.T) {
		c := Constraint{
			SegmentID: 0,
			Property:  "dl_state",
			Operator:  models.ConstraintOperatorEQ,
			Value:     "\"CA\"]", // Invalid "]"
		}
		expr, err := c.ToExpr()
		assert.Error(t, err)
		assert.Nil(t, expr)
	})

	t.Run("parse error - no quotes", func(t *testing.T) {
		c := Constraint{
			SegmentID: 0,
			Property:  "dl_state",
			Operator:  models.ConstraintOperatorEQ,
			Value:     "NY", // Invalid string b/c no ""
		}
		expr, err := c.ToExpr()
		assert.Error(t, err)
		assert.Nil(t, expr)
	})

	t.Run("parse error - no quotes in array", func(t *testing.T) {
		c := Constraint{
			SegmentID: 0,
			Property:  "dl_state",
			Operator:  models.ConstraintOperatorIN,
			Value:     "[NY]", // Invalid string b/c no ""
		}
		expr, err := c.ToExpr()
		assert.Error(t, err)
		assert.Nil(t, expr)
	})

	t.Run("happy code path - single EQ", func(t *testing.T) {
		c := Constraint{
			SegmentID: 0,
			Property:  "dl_state",
			Operator:  models.ConstraintOperatorEQ,
			Value:     "\"CA\"",
		}
		expr, err := c.ToExpr()
		assert.NoError(t, err)
		assert.NotNil(t, expr)
	})

	t.Run("happy code path - IN", func(t *testing.T) {
		c := Constraint{
			SegmentID: 0,
			Property:  "dl_state",
			Operator:  models.ConstraintOperatorIN,
			Value:     `["CA", "NY"]`,
		}
		expr, err := c.ToExpr()
		assert.NoError(t, err)
		assert.NotNil(t, expr)
	})
}

func TestConstraintValidate(t *testing.T) {
	t.Run("empty case", func(t *testing.T) {
		c := Constraint{}
		assert.Error(t, c.Validate())
	})

	t.Run("happy code path", func(t *testing.T) {
		c := Constraint{
			SegmentID: 0,
			Property:  "dl_state",
			Operator:  models.ConstraintOperatorEQ,
			Value:     "\"CA\"",
		}
		assert.NoError(t, c.Validate())
	})
}

func TestConstraintArray(t *testing.T) {
	cs := ConstraintArray{
		{
			SegmentID: 0,
			Property:  "dl_state",
			Operator:  models.ConstraintOperatorIN,
			Value:     `["CA", "NY"]`,
		},
		{
			SegmentID: 0,
			Property:  "state",
			Operator:  models.ConstraintOperatorEQ,
			Value:     `{dl_state}`,
		},
	}
	expr, err := cs.ToExpr()
	assert.NoError(t, err)
	assert.NotNil(t, expr)
}

func TestConstraintToMatchFunc(t *testing.T) {
	t.Run("eq string constraint", func(t *testing.T) {
		c := Constraint{
			Property: "dl_state",
			Operator: models.ConstraintOperatorEQ,
			Value:    `"CA"`,
		}
		matchFunc, err := c.ToMatchFunc()
		assert.Nil(t, err)

		match, err := matchFunc(map[string]interface{}{"dl_state": "CA"})
		assert.Nil(t, err)
		assert.True(t, match)

		match, err = matchFunc(map[string]interface{}{"dl_state": "NY"})
		assert.Nil(t, err)
		assert.False(t, match)

		match, err = matchFunc(map[string]interface{}{"dl_state": 31})
		assert.Nil(t, err)
		assert.False(t, match)
	})
	t.Run("eq numeric constraint", func(t *testing.T) {
		c := Constraint{
			Property: "dl_state",
			Operator: models.ConstraintOperatorEQ,
			Value:    "31",
		}
		matchFunc, err := c.ToMatchFunc()
		assert.Nil(t, err)

		match, err := matchFunc(map[string]interface{}{"dl_state": "CA"})
		assert.Nil(t, err)
		assert.False(t, match)

		match, err = matchFunc(map[string]interface{}{"dl_state": 30.})
		assert.Nil(t, err)
		assert.False(t, match)

		match, err = matchFunc(map[string]interface{}{"dl_state": 31.})
		assert.Nil(t, err)
		assert.True(t, match)
	})
	t.Run("eq bool constraint", func(t *testing.T) {
		c := Constraint{
			Property: "premium",
			Operator: models.ConstraintOperatorEQ,
			Value:    "true",
		}
		matchFunc, err := c.ToMatchFunc()
		assert.Nil(t, err)

		match, err := matchFunc(map[string]interface{}{"premium": true})
		assert.Nil(t, err)
		assert.True(t, match)

		match, err = matchFunc(map[string]interface{}{"premium": false})
		assert.Nil(t, err)
		assert.False(t, match)

		match, err = matchFunc(map[string]interface{}{"premium": "on"})
		assert.Nil(t, err)
		assert.False(t, match)
	})
	t.Run("eq object constraint", func(t *testing.T) {
		c := Constraint{
			Property: "dl_state",
			Operator: models.ConstraintOperatorEQ,
			Value:    `{"number":31,"name":"CA"}`,
		}
		matchFunc, err := c.ToMatchFunc()
		assert.Nil(t, err)

		match, err := matchFunc(map[string]interface{}{"dl_state": map[string]interface{}{"number": 31., "name": "CA"}})
		assert.Nil(t, err)
		assert.True(t, match)

		match, err = matchFunc(map[string]interface{}{"dl_state": map[string]interface{}{"name": "CA", "number": 31.}})
		assert.Nil(t, err)
		assert.True(t, match)

		match, err = matchFunc(map[string]interface{}{"dl_state": map[string]interface{}{"number": 30., "name": "WI"}})
		assert.Nil(t, err)
		assert.False(t, match)

		match, err = matchFunc(map[string]interface{}{"dl_state": 31.})
		assert.Nil(t, err)
		assert.False(t, match)
	})
	t.Run("neq constraint", func(t *testing.T) {
		c := Constraint{
			Property: "dl_state",
			Operator: models.ConstraintOperatorNEQ,
			Value:    `"CA"`,
		}
		matchFunc, err := c.ToMatchFunc()
		assert.Nil(t, err)

		match, err := matchFunc(map[string]interface{}{"dl_state": "CA"})
		assert.Nil(t, err)
		assert.False(t, match)

		match, err = matchFunc(map[string]interface{}{"dl_state": "NY"})
		assert.Nil(t, err)
		assert.True(t, match)

		match, err = matchFunc(map[string]interface{}{"dl_state": 31})
		assert.Nil(t, err)
		assert.True(t, match)
	})
	t.Run("lt constraint", func(t *testing.T) {
		c := Constraint{
			Property: "age",
			Operator: models.ConstraintOperatorLT,
			Value:    "18",
		}
		matchFunc, err := c.ToMatchFunc()
		assert.Nil(t, err)

		match, err := matchFunc(map[string]interface{}{"age": 17.})
		assert.Nil(t, err)
		assert.True(t, match)

		match, err = matchFunc(map[string]interface{}{"age": 18.})
		assert.Nil(t, err)
		assert.False(t, match)

		match, err = matchFunc(map[string]interface{}{"age": 19.})
		assert.Nil(t, err)
		assert.False(t, match)
	})
	t.Run("lte constraint", func(t *testing.T) {
		c := Constraint{
			Property: "age",
			Operator: models.ConstraintOperatorLTE,
			Value:    "18",
		}
		matchFunc, err := c.ToMatchFunc()
		assert.Nil(t, err)

		match, err := matchFunc(map[string]interface{}{"age": 17.})
		assert.Nil(t, err)
		assert.True(t, match)

		match, err = matchFunc(map[string]interface{}{"age": 18.})
		assert.Nil(t, err)
		assert.True(t, match)

		match, err = matchFunc(map[string]interface{}{"age": 19.})
		assert.Nil(t, err)
		assert.False(t, match)
	})
	t.Run("gt constraint", func(t *testing.T) {
		c := Constraint{
			Property: "age",
			Operator: models.ConstraintOperatorGT,
			Value:    "18",
		}
		matchFunc, err := c.ToMatchFunc()
		assert.Nil(t, err)

		match, err := matchFunc(map[string]interface{}{"age": 17.})
		assert.Nil(t, err)
		assert.False(t, match)

		match, err = matchFunc(map[string]interface{}{"age": 18.})
		assert.Nil(t, err)
		assert.False(t, match)

		match, err = matchFunc(map[string]interface{}{"age": 19.})
		assert.Nil(t, err)
		assert.True(t, match)
	})
	t.Run("gte constraint", func(t *testing.T) {
		c := Constraint{
			Property: "age",
			Operator: models.ConstraintOperatorGTE,
			Value:    "18",
		}
		matchFunc, err := c.ToMatchFunc()
		assert.Nil(t, err)

		match, err := matchFunc(map[string]interface{}{"age": 17.})
		assert.Nil(t, err)
		assert.False(t, match)

		match, err = matchFunc(map[string]interface{}{"age": 18.})
		assert.Nil(t, err)
		assert.True(t, match)

		match, err = matchFunc(map[string]interface{}{"age": 19.})
		assert.Nil(t, err)
		assert.True(t, match)
	})
	t.Run("ereg constraint", func(t *testing.T) {
		c := Constraint{
			Property: "email",
			Operator: models.ConstraintOperatorEREG,
			Value:    `"^.+@example\\.com$"`,
		}
		matchFunc, err := c.ToMatchFunc()
		assert.Nil(t, err)

		match, err := matchFunc(map[string]interface{}{"email": "foo@example.com"})
		assert.Nil(t, err)
		assert.True(t, match)

		match, err = matchFunc(map[string]interface{}{"email": "bar@example.com"})
		assert.Nil(t, err)
		assert.True(t, match)

		match, err = matchFunc(map[string]interface{}{"email": "foo@mail.com"})
		assert.Nil(t, err)
		assert.False(t, match)
	})
	t.Run("nereg constraint", func(t *testing.T) {
		c := Constraint{
			Property: "email",
			Operator: models.ConstraintOperatorNEREG,
			Value:    `"^.+@example\\.com$"`,
		}
		matchFunc, err := c.ToMatchFunc()
		assert.Nil(t, err)

		match, err := matchFunc(map[string]interface{}{"email": "foo@example.com"})
		assert.Nil(t, err)
		assert.False(t, match)

		match, err = matchFunc(map[string]interface{}{"email": "bar@example.com"})
		assert.Nil(t, err)
		assert.False(t, match)

		match, err = matchFunc(map[string]interface{}{"email": "foo@mail.com"})
		assert.Nil(t, err)
		assert.True(t, match)
	})
	t.Run("in constraint", func(t *testing.T) {
		c := Constraint{
			Property: "tag",
			Operator: models.ConstraintOperatorIN,
			Value:    `["foo", "bar"]`,
		}
		matchFunc, err := c.ToMatchFunc()
		assert.Nil(t, err)

		match, err := matchFunc(map[string]interface{}{"tag": "foo"})
		assert.Nil(t, err)
		assert.True(t, match)

		match, err = matchFunc(map[string]interface{}{"tag": "bar"})
		assert.Nil(t, err)
		assert.True(t, match)

		match, err = matchFunc(map[string]interface{}{"tag": "baz"})
		assert.Nil(t, err)
		assert.False(t, match)

		match, err = matchFunc(map[string]interface{}{"tag": 42})
		assert.Nil(t, err)
		assert.False(t, match)
	})
	t.Run("notin constraint", func(t *testing.T) {
		c := Constraint{
			Property: "tag",
			Operator: models.ConstraintOperatorNOTIN,
			Value:    `["foo", "bar"]`,
		}
		matchFunc, err := c.ToMatchFunc()
		assert.Nil(t, err)

		match, err := matchFunc(map[string]interface{}{"tag": "foo"})
		assert.Nil(t, err)
		assert.False(t, match)

		match, err = matchFunc(map[string]interface{}{"tag": "bar"})
		assert.Nil(t, err)
		assert.False(t, match)

		match, err = matchFunc(map[string]interface{}{"tag": "baz"})
		assert.Nil(t, err)
		assert.True(t, match)

		match, err = matchFunc(map[string]interface{}{"tag": 42})
		assert.Nil(t, err)
		assert.True(t, match)
	})
	t.Run("contains constraint", func(t *testing.T) {
		c := Constraint{
			Property: "tags",
			Operator: models.ConstraintOperatorCONTAINS,
			Value:    `"foo"`,
		}
		matchFunc, err := c.ToMatchFunc()
		assert.Nil(t, err)

		match, err := matchFunc(map[string]interface{}{"tags": []interface{}{"foo", "bar"}})
		assert.Nil(t, err)
		assert.True(t, match)

		match, err = matchFunc(map[string]interface{}{"tags": []interface{}{"foo", "baz"}})
		assert.Nil(t, err)
		assert.True(t, match)

		match, err = matchFunc(map[string]interface{}{"tags": []interface{}{"bar", "baz"}})
		assert.Nil(t, err)
		assert.False(t, match)
	})
	t.Run("notcontains constraint", func(t *testing.T) {
		c := Constraint{
			Property: "tags",
			Operator: models.ConstraintOperatorNOTCONTAINS,
			Value:    `"foo"`,
		}
		matchFunc, err := c.ToMatchFunc()
		assert.Nil(t, err)

		match, err := matchFunc(map[string]interface{}{"tags": []interface{}{"foo", "bar"}})
		assert.Nil(t, err)
		assert.False(t, match)

		match, err = matchFunc(map[string]interface{}{"tags": []interface{}{"foo", "baz"}})
		assert.Nil(t, err)
		assert.False(t, match)

		match, err = matchFunc(map[string]interface{}{"tags": []interface{}{"bar", "baz"}})
		assert.Nil(t, err)
		assert.True(t, match)
	})
}
