package entity

import (
	"github.com/openflagr/flagr/swagger_gen/models"

	"gorm.io/gorm"
)

// GenFixtureFlag is a fixture
func GenFixtureFlag() Flag {
	f := Flag{
		Model:       gorm.Model{ID: 100},
		Key:         "flag_key_100",
		Description: "",
		Enabled:     true,
		Segments:    []Segment{GenFixtureSegment()},
		Variants: []Variant{
			{
				Model:  gorm.Model{ID: 300},
				FlagID: 100,
				Key:    "control",
			},
			{
				Model:  gorm.Model{ID: 301},
				FlagID: 100,
				Key:    "treatment",
				Attachment: map[string]interface{}{
					"value": "321",
				},
			},
		},
		Tags: []Tag{
			{
				Value: "tag1",
			},
			{
				Value: "tag2",
			},
		},
	}
	f.PrepareEvaluation()
	return f
}

func GenFixtureComplexFlag() Flag {
	f := Flag{
		Model:       gorm.Model{ID: 100},
		Key:         "flag_key_100",
		Description: "",
		Enabled:     true,
		Segments:    []Segment{GenFixtureComplexSegment()},
		Variants: []Variant{
			{
				Model:  gorm.Model{ID: 300},
				FlagID: 100,
				Key:    "control",
			},
			{
				Model:  gorm.Model{ID: 301},
				FlagID: 100,
				Key:    "treatment",
				Attachment: map[string]interface{}{
					"value": "321",
				},
			},
		},
		Tags: []Tag{
			{
				Value: "tag1",
			},
			{
				Value: "tag2",
			},
		},
	}
	err := f.PrepareEvaluation()
	if err != nil {
		panic(err)
	}
	return f
}

// GenFixtureSegment is a fixture
func GenFixtureSegment() Segment {
	s := Segment{
		Model:          gorm.Model{ID: 200},
		FlagID:         100,
		Description:    "",
		Rank:           0,
		RolloutPercent: 100,
		Constraints: []Constraint{
			{
				Model:     gorm.Model{ID: 500},
				SegmentID: 200,
				Property:  "dl_state",
				Operator:  models.ConstraintOperatorEQ,
				Value:     `"CA"`,
			},
		},
		Distributions: []Distribution{
			{
				Model:      gorm.Model{ID: 400},
				SegmentID:  200,
				VariantID:  300,
				VariantKey: "control",
				Percent:    50,
			},
			{
				Model:      gorm.Model{ID: 401},
				SegmentID:  200,
				VariantID:  301,
				VariantKey: "treatment",
				Percent:    50,
			},
		},
	}
	s.PrepareEvaluation()
	return s
}

func GenFixtureComplexSegment() Segment {
	s := Segment{
		Model:          gorm.Model{ID: 200},
		FlagID:         100,
		Description:    "",
		Rank:           0,
		RolloutPercent: 100,
		Constraints: []Constraint{
			{
				Model:     gorm.Model{ID: 500},
				SegmentID: 200,
				Property:  "dl_state",
				Operator:  models.ConstraintOperatorEQ,
				Value:     `"CA"`,
			},
			{
				Model:     gorm.Model{ID: 501},
				SegmentID: 200,
				Property:  "dl_state",
				Operator:  models.ConstraintOperatorNEQ,
				Value:     `"NY"`,
			},
			{
				Model:     gorm.Model{ID: 502},
				SegmentID: 200,
				Property:  "age",
				Operator:  models.ConstraintOperatorLT,
				Value:     "100",
			},
			{
				Model:     gorm.Model{ID: 503},
				SegmentID: 200,
				Property:  "age",
				Operator:  models.ConstraintOperatorLTE,
				Value:     "100",
			},
			{
				Model:     gorm.Model{ID: 504},
				SegmentID: 200,
				Property:  "age",
				Operator:  models.ConstraintOperatorGT,
				Value:     "0",
			},
			{
				Model:     gorm.Model{ID: 505},
				SegmentID: 200,
				Property:  "age",
				Operator:  models.ConstraintOperatorGTE,
				Value:     "0",
			},
			{
				Model:     gorm.Model{ID: 506},
				SegmentID: 200,
				Property:  "email",
				Operator:  models.ConstraintOperatorEREG,
				Value:     `".+@example.com"`,
			},
			{
				Model:     gorm.Model{ID: 507},
				SegmentID: 200,
				Property:  "email",
				Operator:  models.ConstraintOperatorNEREG,
				Value:     `".+@mail.com"`,
			},
			{
				Model:     gorm.Model{ID: 508},
				SegmentID: 200,
				Property:  "tag",
				Operator:  models.ConstraintOperatorIN,
				Value:     `["alpha", "beta"]`,
			},
			{
				Model:     gorm.Model{ID: 509},
				SegmentID: 200,
				Property:  "tag",
				Operator:  models.ConstraintOperatorNOTIN,
				Value:     `["gamma", "delta"]`,
			},
			{
				Model:     gorm.Model{ID: 510},
				SegmentID: 200,
				Property:  "versions",
				Operator:  models.ConstraintOperatorCONTAINS,
				Value:     "3",
			},
			{
				Model:     gorm.Model{ID: 511},
				SegmentID: 200,
				Property:  "versions",
				Operator:  models.ConstraintOperatorNOTCONTAINS,
				Value:     "6",
			},
		},
		Distributions: []Distribution{
			{
				Model:      gorm.Model{ID: 400},
				SegmentID:  200,
				VariantID:  300,
				VariantKey: "control",
				Percent:    100,
			},
		},
	}
	err := s.PrepareEvaluation()
	if err != nil {
		panic(err)
	}
	return s
}
