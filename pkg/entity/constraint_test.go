package entity

import (
	"testing"

	"github.com/openflagr/flagr/swagger_gen/models"
	"github.com/stretchr/testify/assert"
)

// TestConstraintToExpr has been removed since we no longer use ToExpr
// The functionality is now tested in TestConstraintPredicate

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

// TestConstraintArray has been removed since we no longer use ToExpr
// The functionality is now tested in TestConstraintArrayPredicateAll

func TestConstraintPredicate(t *testing.T) {
	t.Run("empty case", func(t *testing.T) {
		c := Constraint{}
		match, err := c.Predicate(map[string]interface{}{})
		assert.Error(t, err)
		assert.False(t, match)
	})

	t.Run("invalid entity context", func(t *testing.T) {
		c := Constraint{
			Property: "dl_state",
			Operator: models.ConstraintOperatorEQ,
			Value:    "\"CA\"",
		}
		match, err := c.Predicate("not a map")
		assert.Error(t, err)
		assert.False(t, match)
	})

	t.Run("property not found", func(t *testing.T) {
		c := Constraint{
			Property: "dl_state",
			Operator: models.ConstraintOperatorEQ,
			Value:    "\"CA\"",
		}
		match, err := c.Predicate(map[string]interface{}{
			"other_property": "value",
		})
		assert.NoError(t, err)
		assert.False(t, match)
	})

	t.Run("EQ operator - match", func(t *testing.T) {
		c := Constraint{
			Property: "dl_state",
			Operator: models.ConstraintOperatorEQ,
			Value:    "\"CA\"",
		}
		match, err := c.Predicate(map[string]interface{}{
			"dl_state": "CA",
		})
		assert.NoError(t, err)
		assert.True(t, match)
	})

	t.Run("EQ operator - no match", func(t *testing.T) {
		c := Constraint{
			Property: "dl_state",
			Operator: models.ConstraintOperatorEQ,
			Value:    "\"CA\"",
		}
		match, err := c.Predicate(map[string]interface{}{
			"dl_state": "NY",
		})
		assert.NoError(t, err)
		assert.False(t, match)
	})

	t.Run("NEQ operator - match", func(t *testing.T) {
		c := Constraint{
			Property: "dl_state",
			Operator: models.ConstraintOperatorNEQ,
			Value:    "\"CA\"",
		}
		match, err := c.Predicate(map[string]interface{}{
			"dl_state": "NY",
		})
		assert.NoError(t, err)
		assert.True(t, match)
	})

	t.Run("LT operator - match", func(t *testing.T) {
		c := Constraint{
			Property: "age",
			Operator: models.ConstraintOperatorLT,
			Value:    "30",
		}
		match, err := c.Predicate(map[string]interface{}{
			"age": 25,
		})
		assert.NoError(t, err)
		assert.True(t, match)
	})

	t.Run("IN operator - match", func(t *testing.T) {
		c := Constraint{
			Property: "dl_state",
			Operator: models.ConstraintOperatorIN,
			Value:    `["CA", "NY"]`,
		}
		match, err := c.Predicate(map[string]interface{}{
			"dl_state": "CA",
		})
		assert.NoError(t, err)
		assert.True(t, match)
	})

	t.Run("IN operator - no match", func(t *testing.T) {
		c := Constraint{
			Property: "dl_state",
			Operator: models.ConstraintOperatorIN,
			Value:    `["CA", "NY"]`,
		}
		match, err := c.Predicate(map[string]interface{}{
			"dl_state": "TX",
		})
		assert.NoError(t, err)
		assert.False(t, match)
	})

	t.Run("CONTAINS operator - match", func(t *testing.T) {
		c := Constraint{
			Property: "tags",
			Operator: models.ConstraintOperatorCONTAINS,
			Value:    "\"premium\"",
		}
		match, err := c.Predicate(map[string]interface{}{
			"tags": []string{"free", "premium", "trial"},
		})
		assert.NoError(t, err)
		assert.True(t, match)
	})
}

func TestConstraintArrayPredicateAll(t *testing.T) {
	t.Run("empty array", func(t *testing.T) {
		cs := ConstraintArray{}
		match, err := cs.PredicateAll(map[string]interface{}{})
		assert.NoError(t, err)
		assert.True(t, match)
	})

	t.Run("all constraints match", func(t *testing.T) {
		cs := ConstraintArray{
			{
				Property: "dl_state",
				Operator: models.ConstraintOperatorEQ,
				Value:    "\"CA\"",
			},
			{
				Property: "age",
				Operator: models.ConstraintOperatorGTE,
				Value:    "21",
			},
		}
		match, err := cs.PredicateAll(map[string]interface{}{
			"dl_state": "CA",
			"age":      25,
		})
		assert.NoError(t, err)
		assert.True(t, match)
	})

	t.Run("one constraint doesn't match", func(t *testing.T) {
		cs := ConstraintArray{
			{
				Property: "dl_state",
				Operator: models.ConstraintOperatorEQ,
				Value:    "\"CA\"",
			},
			{
				Property: "age",
				Operator: models.ConstraintOperatorGTE,
				Value:    "21",
			},
		}
		match, err := cs.PredicateAll(map[string]interface{}{
			"dl_state": "NY",
			"age":      25,
		})
		assert.NoError(t, err)
		assert.False(t, match)
	})

	t.Run("property missing", func(t *testing.T) {
		cs := ConstraintArray{
			{
				Property: "dl_state",
				Operator: models.ConstraintOperatorEQ,
				Value:    "\"CA\"",
			},
			{
				Property: "age",
				Operator: models.ConstraintOperatorGTE,
				Value:    "21",
			},
		}
		match, err := cs.PredicateAll(map[string]interface{}{
			"age": 25,
			// dl_state is missing
		})
		assert.NoError(t, err)
		assert.False(t, match)
	})
}
