package entity

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSegmentPrepareEvaluation(t *testing.T) {
	t.Run("happy code path", func(t *testing.T) {
		s := GenFixtureSegment()
		assert.NoError(t, s.PrepareEvaluation())
		assert.NotNil(t, s.SegmentEvaluation.DistributionArray)
	})

	t.Run("distribution array setup", func(t *testing.T) {
		s := GenFixtureSegment()
		s.SegmentEvaluation = SegmentEvaluation{}
		s.Constraints[0].Value = `"CA"]` // invalid value, but doesn't matter for PrepareEvaluation now
		assert.NoError(t, s.PrepareEvaluation())
		assert.Equal(t, []uint{300, 301}, s.SegmentEvaluation.DistributionArray.VariantIDs)
		assert.Equal(t, []int{500, 1000}, s.SegmentEvaluation.DistributionArray.PercentsAccumulated)
	})
}

func TestSegmentPreload(t *testing.T) {
	t.Run("happy code path", func(t *testing.T) {
		s := GenFixtureSegment()
		f := GenFixtureFlag()
		db := PopulateTestDB(f)

		tmpDB, dbErr := db.DB()
		if dbErr != nil {
			t.Errorf("Failed to get database")
		}

		defer tmpDB.Close()

		err := s.Preload(db)
		assert.NoError(t, err)
	})
}

func TestSegmentEvaluateConstraints(t *testing.T) {
	t.Run("matching entity context", func(t *testing.T) {
		s := GenFixtureSegment()
		entityContext := map[string]interface{}{
			"dl_state": "CA",
		}
		match, err := s.EvaluateConstraints(entityContext)
		assert.NoError(t, err)
		assert.True(t, match)
	})

	t.Run("non-matching entity context", func(t *testing.T) {
		s := GenFixtureSegment()
		entityContext := map[string]interface{}{
			"dl_state": "NY",
		}
		match, err := s.EvaluateConstraints(entityContext)
		assert.NoError(t, err)
		assert.False(t, match)
	})

	t.Run("invalid entity context", func(t *testing.T) {
		s := GenFixtureSegment()
		entityContext := "not a map"
		match, err := s.EvaluateConstraints(entityContext)
		assert.Error(t, err)
		assert.False(t, match)
	})

	t.Run("no constraints", func(t *testing.T) {
		s := GenFixtureSegment()
		s.Constraints = nil
		s.PrepareEvaluation()
		entityContext := map[string]interface{}{}
		match, err := s.EvaluateConstraints(entityContext)
		assert.NoError(t, err)
		assert.True(t, match)
	})
}
