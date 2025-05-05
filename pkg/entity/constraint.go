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

// Constraint is the unit of constraints
type Constraint struct {
	gorm.Model

	SegmentID uint `gorm:"index:idx_constraint_segmentid"`
	Property  string
	Operator  string
	Value     string `gorm:"type:text"`
}

// ConstraintArray is an array of Constraint
type ConstraintArray []Constraint

// OperatorToExprMap maps from the swagger model operator to condition operator
var OperatorToExprMap = map[string]string{
	models.ConstraintOperatorEQ:          "==",
	models.ConstraintOperatorNEQ:         "!=",
	models.ConstraintOperatorLT:          "<",
	models.ConstraintOperatorLTE:         "<=",
	models.ConstraintOperatorGT:          ">",
	models.ConstraintOperatorGTE:         ">=",
	models.ConstraintOperatorEREG:        "=~",
	models.ConstraintOperatorNEREG:       "!~",
	models.ConstraintOperatorIN:          "IN",
	models.ConstraintOperatorNOTIN:       "NOT IN",
	models.ConstraintOperatorCONTAINS:    "CONTAINS",
	models.ConstraintOperatorNOTCONTAINS: "NOT CONTAINS",
}

// Validate validates Constraint
func (c *Constraint) Validate() error {
	if c.Property == "" || c.Operator == "" || c.Value == "" {
		return fmt.Errorf(
			"empty Property/Operator/Value: %s/%s/%s",
			c.Property,
			c.Operator,
			c.Value,
		)
	}

	// Check if the operator is valid
	_, ok := OperatorToExprMap[c.Operator]
	if !ok {
		return fmt.Errorf("not supported operator: %s", c.Operator)
	}

	// For operators that expect JSON arrays or objects, validate the JSON
	switch c.Operator {
	case models.ConstraintOperatorIN, models.ConstraintOperatorNOTIN:
		var values []interface{}
		if err := json.Unmarshal([]byte(c.Value), &values); err != nil {
			return fmt.Errorf("invalid array value for IN/NOT IN operator: %s", err)
		}
	}

	return nil
}

// Predicate evaluates the constraint against the entity context directly
// without using the auxiliary Expr representation
func (c *Constraint) Predicate(entityContext interface{}) (bool, error) {
	if c.Property == "" || c.Operator == "" || c.Value == "" {
		return false, fmt.Errorf(
			"empty Property/Operator/Value: %s/%s/%s",
			c.Property,
			c.Operator,
			c.Value,
		)
	}

	m, ok := entityContext.(map[string]interface{})
	if !ok {
		return false, fmt.Errorf("invalid entity_context: %v", entityContext)
	}

	// Get the property value from the entity context
	propValue, exists := m[c.Property]
	if !exists {
		// If property doesn't exist, it doesn't match
		return false, nil
	}

	// Parse the constraint value based on the operator
	var constraintValue interface{}

	// For operators that expect JSON arrays or objects, parse the value
	switch c.Operator {
	case models.ConstraintOperatorIN, models.ConstraintOperatorNOTIN:
		// Parse JSON array
		var values []interface{}
		if err := json.Unmarshal([]byte(c.Value), &values); err != nil {
			return false, fmt.Errorf("invalid array value for IN/NOT IN operator: %s", err)
		}
		constraintValue = values
	default:
		// For other operators, use the raw value but remove quotes if it's a string
		constraintValue = strings.Trim(c.Value, "\"")
	}

	// Apply the operator
	switch c.Operator {
	case models.ConstraintOperatorEQ:
		return reflect.DeepEqual(propValue, constraintValue), nil
	case models.ConstraintOperatorNEQ:
		return !reflect.DeepEqual(propValue, constraintValue), nil
	case models.ConstraintOperatorLT:
		return compareValues(propValue, constraintValue) < 0, nil
	case models.ConstraintOperatorLTE:
		return compareValues(propValue, constraintValue) <= 0, nil
	case models.ConstraintOperatorGT:
		return compareValues(propValue, constraintValue) > 0, nil
	case models.ConstraintOperatorGTE:
		return compareValues(propValue, constraintValue) >= 0, nil
	case models.ConstraintOperatorEREG:
		re, err := regexp.Compile(constraintValue.(string))
		if err != nil {
			return false, fmt.Errorf("invalid regex: %s", err)
		}
		return re.MatchString(fmt.Sprintf("%v", propValue)), nil
	case models.ConstraintOperatorNEREG:
		re, err := regexp.Compile(constraintValue.(string))
		if err != nil {
			return false, fmt.Errorf("invalid regex: %s", err)
		}
		return !re.MatchString(fmt.Sprintf("%v", propValue)), nil
	case models.ConstraintOperatorIN:
		return contains(constraintValue.([]interface{}), propValue), nil
	case models.ConstraintOperatorNOTIN:
		return !contains(constraintValue.([]interface{}), propValue), nil
	case models.ConstraintOperatorCONTAINS:
		return containsElement(propValue, constraintValue), nil
	case models.ConstraintOperatorNOTCONTAINS:
		return !containsElement(propValue, constraintValue), nil
	default:
		return false, fmt.Errorf("unsupported operator: %s", c.Operator)
	}
}

// Helper function to compare values for inequality operators
func compareValues(a, b interface{}) int {
	// Convert to comparable types if needed
	aFloat, aOk := toFloat64(a)
	bFloat, bOk := toFloat64(b)

	if aOk && bOk {
		// Both are numeric, compare as numbers
		if aFloat < bFloat {
			return -1
		} else if aFloat > bFloat {
			return 1
		}
		return 0
	}

	// Compare as strings
	aStr := fmt.Sprintf("%v", a)
	bStr := fmt.Sprintf("%v", b)

	if aStr < bStr {
		return -1
	} else if aStr > bStr {
		return 1
	}
	return 0
}

// Helper function to check if a slice contains an element
func contains(slice []interface{}, element interface{}) bool {
	for _, item := range slice {
		if reflect.DeepEqual(item, element) {
			return true
		}
	}
	return false
}

// Helper function to check if an element contains another element
func containsElement(container, element interface{}) bool {
	// If container is a string, check if it contains the element as a substring
	containerStr, containerIsStr := container.(string)
	elementStr, elementIsStr := element.(string)
	if containerIsStr && elementIsStr {
		return strings.Contains(containerStr, elementStr)
	}

	// If container is a slice or array, check if it contains the element
	containerVal := reflect.ValueOf(container)
	if containerVal.Kind() == reflect.Slice || containerVal.Kind() == reflect.Array {
		for i := 0; i < containerVal.Len(); i++ {
			if reflect.DeepEqual(containerVal.Index(i).Interface(), element) {
				return true
			}
		}
	}

	return false
}

// PredicateAll evaluates all constraints against the entity context directly
// without using the auxiliary Expr representation
// All constraints are joined with AND logic
func (cs ConstraintArray) PredicateAll(entityContext interface{}) (bool, error) {
	if len(cs) == 0 {
		return true, nil
	}

	for _, c := range cs {
		match, err := c.Predicate(entityContext)
		if err != nil {
			return false, err
		}
		if !match {
			// If any constraint doesn't match, the whole array doesn't match
			return false, nil
		}
	}

	// All constraints matched
	return true, nil
}
