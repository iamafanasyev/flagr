package entity

import (
	"encoding/json"
	"fmt"
	"reflect"
	"regexp"
	"strings"

	"github.com/openflagr/flagr/swagger_gen/models"
	"gorm.io/gorm"
)

// SegmentDefaultRank is the default rank when we create the segment
const SegmentDefaultRank = uint(999)

// Segment is the unit of segmentation
type Segment struct {
	gorm.Model
	FlagID         uint   `gorm:"index:idx_segment_flagid"`
	Description    string `gorm:"type:text"`
	Rank           uint
	RolloutPercent uint
	Constraints    ConstraintArray
	Distributions  []Distribution

	// Purely for evaluation

	SegmentEvaluation SegmentEvaluation `gorm:"-" json:"-"`
}

// EvaluateConstraints evaluates the segment's constraints against the entity context
// Returns true if the constraints match, false otherwise
func (s *Segment) EvaluateConstraints(entityContext interface{}) (bool, error) {
	if len(s.Constraints) == 0 {
		return true, nil
	}

	// Use the PredicateAll function directly on the constraints
	return s.Constraints.PredicateAll(entityContext)
}

// PreloadConstraintsDistribution preloads constraints and distributions
// for segment
func PreloadConstraintsDistribution(db *gorm.DB) *gorm.DB {
	return db.
		Preload("Distributions", func(db *gorm.DB) *gorm.DB {
			return db.Order("variant_id")
		}).
		Preload("Constraints", func(db *gorm.DB) *gorm.DB {
			return db.Order("created_at")
		})
}

// Preload preloads the segment
func (s *Segment) Preload(db *gorm.DB) error {
	return PreloadConstraintsDistribution(db).First(s, s.Model.ID).Error
}

// SegmentEvaluation is a struct that holds the necessary info for evaluation
type SegmentEvaluation struct {
	DistributionArray DistributionArray
	Check             func(entityContext interface{}) (bool, error)
}

// PrepareEvaluation prepares the segment for evaluation by denormalizing distributions
func (s *Segment) PrepareEvaluation() error {
	dLen := len(s.Distributions)

	// Pre-compile constraint values
	type preCompiledConstraint struct {
		property      string
		operator      string
		value         interface{}
		compiledRegex *regexp.Regexp
		numericValue  float64
		isNumeric     bool
		checkFunc     func(propValue interface{}) bool
	}

	preCompiled := make([]preCompiledConstraint, 0, len(s.Constraints))

	for _, c := range s.Constraints {
		pc := preCompiledConstraint{
			property: c.Property,
			operator: c.Operator,
		}

		// Pre-parse constraint value based on operator
		switch c.Operator {
		case models.ConstraintOperatorIN, models.ConstraintOperatorNOTIN:
			// Parse JSON array once
			var values []interface{}
			if err := json.Unmarshal([]byte(c.Value), &values); err != nil {
				return fmt.Errorf("invalid array value for IN/NOT IN operator: %s", err)
			}
			pc.value = values

			// Create check function
			if c.Operator == models.ConstraintOperatorIN {
				pc.checkFunc = func(propValue interface{}) bool {
					return contains(values, propValue)
				}
			} else { // NOT IN
				pc.checkFunc = func(propValue interface{}) bool {
					return !contains(values, propValue)
				}
			}

		case models.ConstraintOperatorEREG, models.ConstraintOperatorNEREG:
			// Compile regex once
			regex, err := regexp.Compile(strings.Trim(c.Value, "\""))
			if err != nil {
				return fmt.Errorf("invalid regex: %s", err)
			}
			pc.compiledRegex = regex

			// Create check function
			if c.Operator == models.ConstraintOperatorEREG {
				pc.checkFunc = func(propValue interface{}) bool {
					return regex.MatchString(fmt.Sprintf("%v", propValue))
				}
			} else { // NEREG
				pc.checkFunc = func(propValue interface{}) bool {
					return !regex.MatchString(fmt.Sprintf("%v", propValue))
				}
			}

		case models.ConstraintOperatorLT, models.ConstraintOperatorLTE, models.ConstraintOperatorGT, models.ConstraintOperatorGTE:
			// For comparison operators, try to parse as numeric value
			strValue := strings.Trim(c.Value, "\"")
			pc.value = strValue

			// Try to parse as numeric
			numValue, ok := toFloat64(strValue)
			if ok {
				pc.numericValue = numValue
				pc.isNumeric = true

				// Create check function with numeric comparison
				switch c.Operator {
				case models.ConstraintOperatorLT:
					pc.checkFunc = func(propValue interface{}) bool {
						propNumeric, propOk := toFloat64(propValue)
						if propOk {
							return propNumeric < numValue
						}
						return compareValues(propValue, strValue) < 0
					}
				case models.ConstraintOperatorLTE:
					pc.checkFunc = func(propValue interface{}) bool {
						propNumeric, propOk := toFloat64(propValue)
						if propOk {
							return propNumeric <= numValue
						}
						return compareValues(propValue, strValue) <= 0
					}
				case models.ConstraintOperatorGT:
					pc.checkFunc = func(propValue interface{}) bool {
						propNumeric, propOk := toFloat64(propValue)
						if propOk {
							return propNumeric > numValue
						}
						return compareValues(propValue, strValue) > 0
					}
				case models.ConstraintOperatorGTE:
					pc.checkFunc = func(propValue interface{}) bool {
						propNumeric, propOk := toFloat64(propValue)
						if propOk {
							return propNumeric >= numValue
						}
						return compareValues(propValue, strValue) >= 0
					}
				}
			} else {
				// Create check function with string comparison
				switch c.Operator {
				case models.ConstraintOperatorLT:
					pc.checkFunc = func(propValue interface{}) bool {
						return compareValues(propValue, strValue) < 0
					}
				case models.ConstraintOperatorLTE:
					pc.checkFunc = func(propValue interface{}) bool {
						return compareValues(propValue, strValue) <= 0
					}
				case models.ConstraintOperatorGT:
					pc.checkFunc = func(propValue interface{}) bool {
						return compareValues(propValue, strValue) > 0
					}
				case models.ConstraintOperatorGTE:
					pc.checkFunc = func(propValue interface{}) bool {
						return compareValues(propValue, strValue) >= 0
					}
				}
			}

		case models.ConstraintOperatorEQ:
			// For equality operators, use the raw value but remove quotes if it's a string
			strValue := strings.Trim(c.Value, "\"")
			pc.value = strValue

			pc.checkFunc = func(propValue interface{}) bool {
				return reflect.DeepEqual(propValue, strValue)
			}

		case models.ConstraintOperatorNEQ:
			// For inequality operators, use the raw value but remove quotes if it's a string
			strValue := strings.Trim(c.Value, "\"")
			pc.value = strValue

			pc.checkFunc = func(propValue interface{}) bool {
				return !reflect.DeepEqual(propValue, strValue)
			}

		case models.ConstraintOperatorCONTAINS:
			// For CONTAINS operator, use the raw value but remove quotes if it's a string
			strValue := strings.Trim(c.Value, "\"")
			pc.value = strValue

			pc.checkFunc = func(propValue interface{}) bool {
				return containsElement(propValue, strValue)
			}

		case models.ConstraintOperatorNOTCONTAINS:
			// For NOT CONTAINS operator, use the raw value but remove quotes if it's a string
			strValue := strings.Trim(c.Value, "\"")
			pc.value = strValue

			pc.checkFunc = func(propValue interface{}) bool {
				return !containsElement(propValue, strValue)
			}

		default:
			// For other operators, use the raw value but remove quotes if it's a string
			strValue := strings.Trim(c.Value, "\"")
			pc.value = strValue
		}

		preCompiled = append(preCompiled, pc)
	}

	se := SegmentEvaluation{
		DistributionArray: DistributionArray{
			VariantIDs:          make([]uint, dLen),
			PercentsAccumulated: make([]int, dLen),
		},
		Check: func(entityContext interface{}) (bool, error) {
			if len(preCompiled) == 0 {
				return true, nil
			}

			m, ok := entityContext.(map[string]interface{})
			if !ok {
				return false, fmt.Errorf("invalid entity_context: %v", entityContext)
			}

			// Evaluate each pre-compiled constraint
			for _, pc := range preCompiled {
				// Get property from context
				propValue, exists := m[pc.property]
				if !exists {
					// If property doesn't exist, it doesn't match
					return false, nil
				}

				// Apply the operator using pre-compiled values
				var match bool
				switch pc.operator {
				case models.ConstraintOperatorEQ:
					match = reflect.DeepEqual(propValue, pc.value)
				case models.ConstraintOperatorNEQ:
					match = !reflect.DeepEqual(propValue, pc.value)
				case models.ConstraintOperatorLT:
					if pc.isNumeric {
						// If constraint value is numeric, try to convert property value to numeric
						propNumeric, propOk := toFloat64(propValue)
						if propOk {
							// Both are numeric, compare directly
							match = propNumeric < pc.numericValue
						} else {
							// Property value is not numeric, fall back to string comparison
							match = compareValues(propValue, pc.value) < 0
						}
					} else {
						// Constraint value is not numeric, use regular comparison
						match = compareValues(propValue, pc.value) < 0
					}
				case models.ConstraintOperatorLTE:
					if pc.isNumeric {
						// If constraint value is numeric, try to convert property value to numeric
						propNumeric, propOk := toFloat64(propValue)
						if propOk {
							// Both are numeric, compare directly
							match = propNumeric <= pc.numericValue
						} else {
							// Property value is not numeric, fall back to string comparison
							match = compareValues(propValue, pc.value) <= 0
						}
					} else {
						// Constraint value is not numeric, use regular comparison
						match = compareValues(propValue, pc.value) <= 0
					}
				case models.ConstraintOperatorGT:
					if pc.isNumeric {
						// If constraint value is numeric, try to convert property value to numeric
						propNumeric, propOk := toFloat64(propValue)
						if propOk {
							// Both are numeric, compare directly
							match = propNumeric > pc.numericValue
						} else {
							// Property value is not numeric, fall back to string comparison
							match = compareValues(propValue, pc.value) > 0
						}
					} else {
						// Constraint value is not numeric, use regular comparison
						match = compareValues(propValue, pc.value) > 0
					}
				case models.ConstraintOperatorGTE:
					if pc.isNumeric {
						// If constraint value is numeric, try to convert property value to numeric
						propNumeric, propOk := toFloat64(propValue)
						if propOk {
							// Both are numeric, compare directly
							match = propNumeric >= pc.numericValue
						} else {
							// Property value is not numeric, fall back to string comparison
							match = compareValues(propValue, pc.value) >= 0
						}
					} else {
						// Constraint value is not numeric, use regular comparison
						match = compareValues(propValue, pc.value) >= 0
					}
				case models.ConstraintOperatorEREG:
					match = pc.compiledRegex.MatchString(fmt.Sprintf("%v", propValue))
				case models.ConstraintOperatorNEREG:
					match = !pc.compiledRegex.MatchString(fmt.Sprintf("%v", propValue))
				case models.ConstraintOperatorIN:
					match = contains(pc.value.([]interface{}), propValue)
				case models.ConstraintOperatorNOTIN:
					match = !contains(pc.value.([]interface{}), propValue)
				case models.ConstraintOperatorCONTAINS:
					match = containsElement(propValue, pc.value)
				case models.ConstraintOperatorNOTCONTAINS:
					match = !containsElement(propValue, pc.value)
				default:
					return false, fmt.Errorf("unsupported operator: %s", pc.operator)
				}

				if !match {
					// If any constraint doesn't match, the whole segment doesn't match
					return false, nil
				}
			}

			// All constraints matched
			return true, nil
		},
	}

	for i, d := range s.Distributions {
		se.DistributionArray.VariantIDs[i] = d.VariantID
		if i == 0 {
			se.DistributionArray.PercentsAccumulated[i] = int(d.Percent * PercentMultiplier)
		} else {
			se.DistributionArray.PercentsAccumulated[i] = se.DistributionArray.PercentsAccumulated[i-1] + int(d.Percent*PercentMultiplier)
		}
	}

	s.SegmentEvaluation = se
	return nil
}
