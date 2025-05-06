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

		match, err = matchFunc(map[string]interface{}{"dl_state": 30})
		assert.Nil(t, err)
		assert.False(t, match)

		match, err = matchFunc(map[string]interface{}{"dl_state": "31"})
		assert.Nil(t, err)
		assert.False(t, match)

		match, err = matchFunc(map[string]interface{}{"dl_state": 31})
		assert.Nil(t, err)
		assert.True(t, match)

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
	t.Run("eq unsupported constraint target value", func(t *testing.T) {
		c := Constraint{
			Property: "dl_state",
			Operator: models.ConstraintOperatorEQ,
			Value:    `{"number":31,"name":"CA"}`,
		}
		_, err := c.ToMatchFunc()
		assert.NotNil(t, err)
	})
	t.Run("neq string constraint", func(t *testing.T) {
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
	t.Run("neq numeric constraint", func(t *testing.T) {
		c := Constraint{
			Property: "dl_state",
			Operator: models.ConstraintOperatorNEQ,
			Value:    "31",
		}
		matchFunc, err := c.ToMatchFunc()
		assert.Nil(t, err)

		match, err := matchFunc(map[string]interface{}{"dl_state": 31})
		assert.Nil(t, err)
		assert.False(t, match)

		match, err = matchFunc(map[string]interface{}{"dl_state": 31.})
		assert.Nil(t, err)
		assert.False(t, match)

		match, err = matchFunc(map[string]interface{}{"dl_state": 30})
		assert.Nil(t, err)
		assert.True(t, match)

		match, err = matchFunc(map[string]interface{}{"dl_state": "31"})
		assert.Nil(t, err)
		assert.True(t, match)
	})
	t.Run("neq bool constraint", func(t *testing.T) {
		c := Constraint{
			Property: "premium",
			Operator: models.ConstraintOperatorNEQ,
			Value:    "true",
		}
		matchFunc, err := c.ToMatchFunc()
		assert.Nil(t, err)

		match, err := matchFunc(map[string]interface{}{"premium": true})
		assert.Nil(t, err)
		assert.False(t, match)

		match, err = matchFunc(map[string]interface{}{"premium": false})
		assert.Nil(t, err)
		assert.True(t, match)

		match, err = matchFunc(map[string]interface{}{"premium": "true"})
		assert.Nil(t, err)
		assert.True(t, match)

		match, err = matchFunc(map[string]interface{}{"premium": "on"})
		assert.Nil(t, err)
		assert.True(t, match)
	})
	t.Run("eq unsupported constraint target value", func(t *testing.T) {
		c := Constraint{
			Property: "dl_state",
			Operator: models.ConstraintOperatorNEQ,
			Value:    `[1, 2, 3]`,
		}
		_, err := c.ToMatchFunc()
		assert.NotNil(t, err)
	})
	t.Run("lt constraint", func(t *testing.T) {
		c := Constraint{
			Property: "age",
			Operator: models.ConstraintOperatorLT,
			Value:    "18",
		}
		matchFunc, err := c.ToMatchFunc()
		assert.Nil(t, err)

		match, err := matchFunc(map[string]interface{}{"age": 17})
		assert.Nil(t, err)
		assert.True(t, match)

		match, err = matchFunc(map[string]interface{}{"age": 17.5})
		assert.Nil(t, err)
		assert.True(t, match)

		match, err = matchFunc(map[string]interface{}{"age": 18})
		assert.Nil(t, err)
		assert.False(t, match)

		match, err = matchFunc(map[string]interface{}{"age": 18.})
		assert.Nil(t, err)
		assert.False(t, match)

		match, err = matchFunc(map[string]interface{}{"age": 19})
		assert.Nil(t, err)
		assert.False(t, match)

		match, err = matchFunc(map[string]interface{}{"age": "unknown"})
		assert.Nil(t, err)
		assert.False(t, match)

		match, err = matchFunc(map[string]interface{}{"age": "17"})
		assert.Nil(t, err)
		assert.False(t, match)
	})
	t.Run("lt unsupported constraint target value", func(t *testing.T) {
		c := Constraint{
			Property: "dl_state",
			Operator: models.ConstraintOperatorLT,
			Value:    `"foobar"`,
		}
		_, err := c.ToMatchFunc()
		assert.NotNil(t, err)

		c = Constraint{
			Property: "dl_state",
			Operator: models.ConstraintOperatorLT,
			Value:    `true`,
		}
		_, err = c.ToMatchFunc()
		assert.NotNil(t, err)

		c = Constraint{
			Property: "dl_state",
			Operator: models.ConstraintOperatorLT,
			Value:    `[1, 2, 3]`,
		}
		_, err = c.ToMatchFunc()
		assert.NotNil(t, err)
	})
	t.Run("lte constraint", func(t *testing.T) {
		c := Constraint{
			Property: "age",
			Operator: models.ConstraintOperatorLTE,
			Value:    "18",
		}
		matchFunc, err := c.ToMatchFunc()
		assert.Nil(t, err)

		match, err := matchFunc(map[string]interface{}{"age": 17})
		assert.Nil(t, err)
		assert.True(t, match)

		match, err = matchFunc(map[string]interface{}{"age": 17.})
		assert.Nil(t, err)
		assert.True(t, match)

		match, err = matchFunc(map[string]interface{}{"age": 18})
		assert.Nil(t, err)
		assert.True(t, match)

		match, err = matchFunc(map[string]interface{}{"age": 18.})
		assert.Nil(t, err)
		assert.True(t, match)

		match, err = matchFunc(map[string]interface{}{"age": 19})
		assert.Nil(t, err)
		assert.False(t, match)

		match, err = matchFunc(map[string]interface{}{"age": 19.})
		assert.Nil(t, err)
		assert.False(t, match)

		match, err = matchFunc(map[string]interface{}{"age": "unknown"})
		assert.Nil(t, err)
		assert.False(t, match)

		match, err = matchFunc(map[string]interface{}{"age": "17"})
		assert.Nil(t, err)
		assert.False(t, match)
	})
	t.Run("lte unsupported constraint target value", func(t *testing.T) {
		c := Constraint{
			Property: "dl_state",
			Operator: models.ConstraintOperatorLTE,
			Value:    `"foobar"`,
		}
		_, err := c.ToMatchFunc()
		assert.NotNil(t, err)

		c = Constraint{
			Property: "dl_state",
			Operator: models.ConstraintOperatorLTE,
			Value:    `true`,
		}
		_, err = c.ToMatchFunc()
		assert.NotNil(t, err)

		c = Constraint{
			Property: "dl_state",
			Operator: models.ConstraintOperatorLTE,
			Value:    `[1, 2, 3]`,
		}
		_, err = c.ToMatchFunc()
		assert.NotNil(t, err)
	})
	t.Run("gt constraint", func(t *testing.T) {
		c := Constraint{
			Property: "age",
			Operator: models.ConstraintOperatorGT,
			Value:    "18",
		}
		matchFunc, err := c.ToMatchFunc()
		assert.Nil(t, err)

		match, err := matchFunc(map[string]interface{}{"age": 17})
		assert.Nil(t, err)
		assert.False(t, match)

		match, err = matchFunc(map[string]interface{}{"age": 17})
		assert.Nil(t, err)
		assert.False(t, match)

		match, err = matchFunc(map[string]interface{}{"age": 18})
		assert.Nil(t, err)
		assert.False(t, match)

		match, err = matchFunc(map[string]interface{}{"age": 18.})
		assert.Nil(t, err)
		assert.False(t, match)

		match, err = matchFunc(map[string]interface{}{"age": 19})
		assert.Nil(t, err)
		assert.True(t, match)

		match, err = matchFunc(map[string]interface{}{"age": 19.})
		assert.Nil(t, err)
		assert.True(t, match)

		match, err = matchFunc(map[string]interface{}{"age": "unknown"})
		assert.Nil(t, err)
		assert.False(t, match)

		match, err = matchFunc(map[string]interface{}{"age": "17"})
		assert.Nil(t, err)
		assert.False(t, match)
	})
	t.Run("gt unsupported constraint target value", func(t *testing.T) {
		c := Constraint{
			Property: "dl_state",
			Operator: models.ConstraintOperatorGT,
			Value:    `"foobar"`,
		}
		_, err := c.ToMatchFunc()
		assert.NotNil(t, err)

		c = Constraint{
			Property: "dl_state",
			Operator: models.ConstraintOperatorGT,
			Value:    `true`,
		}
		_, err = c.ToMatchFunc()
		assert.NotNil(t, err)

		c = Constraint{
			Property: "dl_state",
			Operator: models.ConstraintOperatorGT,
			Value:    `[1, 2, 3]`,
		}
		_, err = c.ToMatchFunc()
		assert.NotNil(t, err)
	})
	t.Run("gte constraint", func(t *testing.T) {
		c := Constraint{
			Property: "age",
			Operator: models.ConstraintOperatorGTE,
			Value:    "18",
		}
		matchFunc, err := c.ToMatchFunc()
		assert.Nil(t, err)

		match, err := matchFunc(map[string]interface{}{"age": 17})
		assert.Nil(t, err)
		assert.False(t, match)

		match, err = matchFunc(map[string]interface{}{"age": 17.})
		assert.Nil(t, err)
		assert.False(t, match)

		match, err = matchFunc(map[string]interface{}{"age": 18})
		assert.Nil(t, err)
		assert.True(t, match)

		match, err = matchFunc(map[string]interface{}{"age": 18.})
		assert.Nil(t, err)
		assert.True(t, match)

		match, err = matchFunc(map[string]interface{}{"age": 19})
		assert.Nil(t, err)
		assert.True(t, match)

		match, err = matchFunc(map[string]interface{}{"age": 19.})
		assert.Nil(t, err)
		assert.True(t, match)

		match, err = matchFunc(map[string]interface{}{"age": "unknown"})
		assert.Nil(t, err)
		assert.False(t, match)

		match, err = matchFunc(map[string]interface{}{"age": "17"})
		assert.Nil(t, err)
		assert.False(t, match)
	})
	t.Run("gt unsupported constraint target value", func(t *testing.T) {
		c := Constraint{
			Property: "dl_state",
			Operator: models.ConstraintOperatorGTE,
			Value:    `"foobar"`,
		}
		_, err := c.ToMatchFunc()
		assert.NotNil(t, err)

		c = Constraint{
			Property: "dl_state",
			Operator: models.ConstraintOperatorGTE,
			Value:    `true`,
		}
		_, err = c.ToMatchFunc()
		assert.NotNil(t, err)

		c = Constraint{
			Property: "dl_state",
			Operator: models.ConstraintOperatorGTE,
			Value:    `[1, 2, 3]`,
		}
		_, err = c.ToMatchFunc()
		assert.NotNil(t, err)
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
	t.Run("ereg unsupported constraint target value", func(t *testing.T) {
		c := Constraint{
			Property: "email",
			Operator: models.ConstraintOperatorEREG,
			Value:    `42`,
		}
		_, err := c.ToMatchFunc()
		assert.NotNil(t, err)

		c = Constraint{
			Property: "email",
			Operator: models.ConstraintOperatorEREG,
			Value:    `true`,
		}
		_, err = c.ToMatchFunc()
		assert.NotNil(t, err)

		c = Constraint{
			Property: "email",
			Operator: models.ConstraintOperatorEREG,
			Value:    `[1, 2, 3]`,
		}
		_, err = c.ToMatchFunc()
		assert.NotNil(t, err)
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
	t.Run("nereg unsupported constraint target value", func(t *testing.T) {
		c := Constraint{
			Property: "email",
			Operator: models.ConstraintOperatorNEREG,
			Value:    `42`,
		}
		_, err := c.ToMatchFunc()
		assert.NotNil(t, err)

		c = Constraint{
			Property: "email",
			Operator: models.ConstraintOperatorNEREG,
			Value:    `true`,
		}
		_, err = c.ToMatchFunc()
		assert.NotNil(t, err)

		c = Constraint{
			Property: "email",
			Operator: models.ConstraintOperatorNEREG,
			Value:    `[1, 2, 3]`,
		}
		_, err = c.ToMatchFunc()
		assert.NotNil(t, err)
	})
	t.Run("in string constraint", func(t *testing.T) {
		c := Constraint{
			Property: "tag",
			Operator: models.ConstraintOperatorIN,
			Value:    `["1", "2"]`,
		}
		matchFunc, err := c.ToMatchFunc()
		assert.Nil(t, err)

		match, err := matchFunc(map[string]interface{}{"tag": "1"})
		assert.Nil(t, err)
		assert.True(t, match)

		match, err = matchFunc(map[string]interface{}{"tag": "2"})
		assert.Nil(t, err)
		assert.True(t, match)

		match, err = matchFunc(map[string]interface{}{"tag": "3"})
		assert.Nil(t, err)
		assert.False(t, match)

		match, err = matchFunc(map[string]interface{}{"tag": 1})
		assert.Nil(t, err)
		assert.False(t, match)

		match, err = matchFunc(map[string]interface{}{"tag": 2})
		assert.Nil(t, err)
		assert.False(t, match)

		match, err = matchFunc(map[string]interface{}{"tag": 3})
		assert.Nil(t, err)
		assert.False(t, match)
	})
	t.Run("in numeric constraint", func(t *testing.T) {
		c := Constraint{
			Property: "version",
			Operator: models.ConstraintOperatorIN,
			Value:    `[1, 2]`,
		}
		matchFunc, err := c.ToMatchFunc()
		assert.Nil(t, err)

		match, err := matchFunc(map[string]interface{}{"version": 1})
		assert.Nil(t, err)
		assert.True(t, match)

		match, err = matchFunc(map[string]interface{}{"version": 1.})
		assert.Nil(t, err)
		assert.True(t, match)

		match, err = matchFunc(map[string]interface{}{"version": 2})
		assert.Nil(t, err)
		assert.True(t, match)

		match, err = matchFunc(map[string]interface{}{"version": 2.})
		assert.Nil(t, err)
		assert.True(t, match)

		match, err = matchFunc(map[string]interface{}{"version": 3})
		assert.Nil(t, err)
		assert.False(t, match)

		match, err = matchFunc(map[string]interface{}{"version": 3.})
		assert.Nil(t, err)
		assert.False(t, match)

		match, err = matchFunc(map[string]interface{}{"version": "1"})
		assert.Nil(t, err)
		assert.False(t, match)

		match, err = matchFunc(map[string]interface{}{"version": "2"})
		assert.Nil(t, err)
		assert.False(t, match)

		match, err = matchFunc(map[string]interface{}{"version": "3"})
		assert.Nil(t, err)
		assert.False(t, match)
	})
	t.Run("in unsupported constraint target value", func(t *testing.T) {
		c := Constraint{
			Property: "version",
			Operator: models.ConstraintOperatorIN,
			Value:    `"42"`,
		}
		_, err := c.ToMatchFunc()
		assert.NotNil(t, err)

		c = Constraint{
			Property: "version",
			Operator: models.ConstraintOperatorIN,
			Value:    `42`,
		}
		_, err = c.ToMatchFunc()
		assert.NotNil(t, err)

		c = Constraint{
			Property: "version",
			Operator: models.ConstraintOperatorIN,
			Value:    `true`,
		}
		_, err = c.ToMatchFunc()
		assert.NotNil(t, err)

		c = Constraint{
			Property: "version",
			Operator: models.ConstraintOperatorIN,
			Value:    `{"values": [1, 2, 3]}`,
		}
		_, err = c.ToMatchFunc()
		assert.NotNil(t, err)

		c = Constraint{
			Property: "version",
			Operator: models.ConstraintOperatorIN,
			Value:    `[true, false]`,
		}
		_, err = c.ToMatchFunc()
		assert.NotNil(t, err)
	})
	t.Run("notin string constraint", func(t *testing.T) {
		c := Constraint{
			Property: "tag",
			Operator: models.ConstraintOperatorNOTIN,
			Value:    `["1", "2"]`,
		}
		matchFunc, err := c.ToMatchFunc()
		assert.Nil(t, err)

		match, err := matchFunc(map[string]interface{}{"tag": "1"})
		assert.Nil(t, err)
		assert.False(t, match)

		match, err = matchFunc(map[string]interface{}{"tag": "2"})
		assert.Nil(t, err)
		assert.False(t, match)

		match, err = matchFunc(map[string]interface{}{"tag": "3"})
		assert.Nil(t, err)
		assert.True(t, match)

		match, err = matchFunc(map[string]interface{}{"tag": 1})
		assert.Nil(t, err)
		assert.True(t, match)

		match, err = matchFunc(map[string]interface{}{"tag": 2})
		assert.Nil(t, err)
		assert.True(t, match)

		match, err = matchFunc(map[string]interface{}{"tag": 3})
		assert.Nil(t, err)
		assert.True(t, match)
	})
	t.Run("notin numeric constraint", func(t *testing.T) {
		c := Constraint{
			Property: "tag",
			Operator: models.ConstraintOperatorNOTIN,
			Value:    `[1, 2]`,
		}
		matchFunc, err := c.ToMatchFunc()
		assert.Nil(t, err)

		match, err := matchFunc(map[string]interface{}{"tag": 1})
		assert.Nil(t, err)
		assert.False(t, match)

		match, err = matchFunc(map[string]interface{}{"tag": 1.})
		assert.Nil(t, err)
		assert.False(t, match)

		match, err = matchFunc(map[string]interface{}{"tag": 2})
		assert.Nil(t, err)
		assert.False(t, match)

		match, err = matchFunc(map[string]interface{}{"tag": 2.})
		assert.Nil(t, err)
		assert.False(t, match)

		match, err = matchFunc(map[string]interface{}{"tag": 3})
		assert.Nil(t, err)
		assert.True(t, match)

		match, err = matchFunc(map[string]interface{}{"tag": "1"})
		assert.Nil(t, err)
		assert.True(t, match)

		match, err = matchFunc(map[string]interface{}{"tag": "2"})
		assert.Nil(t, err)
		assert.True(t, match)

		match, err = matchFunc(map[string]interface{}{"tag": "3"})
		assert.Nil(t, err)
		assert.True(t, match)
	})
	t.Run("notin unsupported constraint target value", func(t *testing.T) {
		c := Constraint{
			Property: "version",
			Operator: models.ConstraintOperatorNOTIN,
			Value:    `"42"`,
		}
		_, err := c.ToMatchFunc()
		assert.NotNil(t, err)

		c = Constraint{
			Property: "version",
			Operator: models.ConstraintOperatorNOTIN,
			Value:    `42`,
		}
		_, err = c.ToMatchFunc()
		assert.NotNil(t, err)

		c = Constraint{
			Property: "version",
			Operator: models.ConstraintOperatorNOTIN,
			Value:    `true`,
		}
		_, err = c.ToMatchFunc()
		assert.NotNil(t, err)

		c = Constraint{
			Property: "version",
			Operator: models.ConstraintOperatorNOTIN,
			Value:    `{"values": [1, 2, 3]}`,
		}
		_, err = c.ToMatchFunc()
		assert.NotNil(t, err)

		c = Constraint{
			Property: "version",
			Operator: models.ConstraintOperatorNOTIN,
			Value:    `[true, false]`,
		}
		_, err = c.ToMatchFunc()
		assert.NotNil(t, err)
	})
	t.Run("contains string constraint", func(t *testing.T) {
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
	t.Run("contains numeric constraint", func(t *testing.T) {
		c := Constraint{
			Property: "tags",
			Operator: models.ConstraintOperatorCONTAINS,
			Value:    `2`,
		}
		matchFunc, err := c.ToMatchFunc()
		assert.Nil(t, err)

		match, err := matchFunc(map[string]interface{}{"tags": []interface{}{1, 2}})
		assert.Nil(t, err)
		assert.True(t, match)

		match, err = matchFunc(map[string]interface{}{"tags": []interface{}{2, 3}})
		assert.Nil(t, err)
		assert.True(t, match)

		match, err = matchFunc(map[string]interface{}{"tags": []interface{}{1, 3}})
		assert.Nil(t, err)
		assert.False(t, match)

		match, err = matchFunc(map[string]interface{}{"tags": []interface{}{"1", "2"}})
		assert.Nil(t, err)
		assert.False(t, match)
	})
	t.Run("contains unsupported constraint target value", func(t *testing.T) {
		c := Constraint{
			Property: "tags",
			Operator: models.ConstraintOperatorCONTAINS,
			Value:    `true`,
		}
		_, err := c.ToMatchFunc()
		assert.NotNil(t, err)

		c = Constraint{
			Property: "tags",
			Operator: models.ConstraintOperatorCONTAINS,
			Value:    `[1, 2, 3]`,
		}
		_, err = c.ToMatchFunc()
		assert.NotNil(t, err)

		c = Constraint{
			Property: "tags",
			Operator: models.ConstraintOperatorCONTAINS,
			Value:    `{"value": 1}`,
		}
		_, err = c.ToMatchFunc()
		assert.NotNil(t, err)
	})
	t.Run("notcontains string constraint", func(t *testing.T) {
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
	t.Run("notcontains numeric constraint", func(t *testing.T) {
		c := Constraint{
			Property: "tags",
			Operator: models.ConstraintOperatorNOTCONTAINS,
			Value:    `2`,
		}
		matchFunc, err := c.ToMatchFunc()
		assert.Nil(t, err)

		match, err := matchFunc(map[string]interface{}{"tags": []interface{}{1, 2}})
		assert.Nil(t, err)
		assert.False(t, match)

		match, err = matchFunc(map[string]interface{}{"tags": []interface{}{2, 3}})
		assert.Nil(t, err)
		assert.False(t, match)

		match, err = matchFunc(map[string]interface{}{"tags": []interface{}{1, 3}})
		assert.Nil(t, err)
		assert.True(t, match)

		match, err = matchFunc(map[string]interface{}{"tags": []interface{}{"1", "2"}})
		assert.Nil(t, err)
		assert.True(t, match)
	})
}
