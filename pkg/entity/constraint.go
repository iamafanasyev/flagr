package entity

import (
	"encoding/json"
	"fmt"
	"reflect"
	"regexp"
	"strconv"
	"strings"

	"github.com/openflagr/flagr/swagger_gen/models"
	"github.com/zhouzhuojie/conditions"
	"gorm.io/gorm"
)

// Constraint is the unit of constraints
type Constraint struct {
	gorm.Model

	SegmentID uint `gorm:"index:idx_constraint_segmentid"`
	Property  string
	Operator  string
	Value     string `gorm:"type:text"`

	// Purely for evaluation
	Match func(entityContext map[string]interface{}) (bool, error) `gorm:"-" json:"-"`
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

// ToExpr transfer the constraint to conditions.Expr for evaluation
func (c *Constraint) ToExpr() (conditions.Expr, error) {
	s, err := c.toExprStr()
	if err != nil {
		return nil, err
	}
	p := conditions.NewParser(strings.NewReader(s))
	expr, err := p.Parse()
	if err != nil {
		return nil, fmt.Errorf("%s. Note: if it's string or array of string, warp it with quotes \"...\"", err)
	}
	return expr, nil
}

func (c *Constraint) toExprStr() (string, error) {
	if c.Property == "" || c.Operator == "" || c.Value == "" {
		return "", fmt.Errorf(
			"empty Property/Operator/Value: %s/%s/%s",
			c.Property,
			c.Operator,
			c.Value,
		)
	}
	o, ok := OperatorToExprMap[c.Operator]
	if !ok {
		return "", fmt.Errorf("not supported operator: %s", c.Operator)
	}

	return fmt.Sprintf("({%s} %s %s)", c.Property, o, c.Value), nil
}

// Validate validates Constraint
func (c *Constraint) Validate() error {
	_, err := c.ToExpr()
	return err
}

// ToExpr maps ConstraintArray to expr by joining 'AND'
func (cs ConstraintArray) ToExpr() (conditions.Expr, error) {
	strs := make([]string, 0, len(cs))
	for _, c := range cs {
		s, err := c.toExprStr()
		if err != nil {
			return nil, err
		}
		strs = append(strs, s)
	}
	exprStr := strings.Join(strs, " AND ")
	p := conditions.NewParser(strings.NewReader(exprStr))
	expr, err := p.Parse()
	if err != nil {
		return nil, fmt.Errorf("%s. Note: if it's string or array of string, wrap it with quotes \"...\"", err)
	}
	return expr, nil
}

func (c *Constraint) PrepareEvaluation() error {
	matchFunc, err := c.ToMatchFunc()
	if err != nil {
		return err
	}
	c.Match = matchFunc
	return nil
}

func (c *Constraint) ToMatchFunc() (func(entityContext map[string]interface{}) (bool, error), error) {
	alwaysFalse := func(entityContext map[string]interface{}) (bool, error) {
		return false, nil
	}
	var propertyPredicate = func(propertyValue interface{}) (bool, error) {
		return false, nil
	}
	switch c.Operator {
	case models.ConstraintOperatorEQ:
		var targetValue interface{}
		if err := json.Unmarshal([]byte(strings.Trim(c.Value, " ")), &targetValue); err != nil {
			return alwaysFalse, fmt.Errorf("invalid constraint value: %s", err)
		}
		propertyPredicate = func(propertyValue interface{}) (bool, error) {
			return reflect.DeepEqual(propertyValue, targetValue), nil
		}
	case models.ConstraintOperatorNEQ:
		var targetValue interface{}
		if err := json.Unmarshal([]byte(strings.Trim(c.Value, " ")), &targetValue); err != nil {
			return alwaysFalse, fmt.Errorf("invalid constraint value: %s", err)
		}
		propertyPredicate = func(propertyValue interface{}) (bool, error) {
			return !reflect.DeepEqual(propertyValue, targetValue), nil
		}
	case models.ConstraintOperatorLT:
		targetValue, err := strconv.ParseFloat(strings.Trim(c.Value, " "), 64)
		if err != nil {
			return alwaysFalse, fmt.Errorf("invalid constraint float value: %s", err)
		}
		propertyPredicate = func(propertyValue interface{}) (bool, error) {
			actualValue, ok := propertyValue.(float64)
			if !ok {
				return false, nil
			}
			return actualValue < targetValue, nil
		}
	case models.ConstraintOperatorLTE:
		targetValue, err := strconv.ParseFloat(strings.Trim(c.Value, " "), 64)
		if err != nil {
			return alwaysFalse, fmt.Errorf("invalid constraint float value: %s", err)
		}
		propertyPredicate = func(propertyValue interface{}) (bool, error) {
			actualValue, ok := propertyValue.(float64)
			if !ok {
				return false, nil
			}
			return actualValue <= targetValue, nil
		}
	case models.ConstraintOperatorGT:
		targetValue, err := strconv.ParseFloat(strings.Trim(c.Value, " "), 64)
		if err != nil {
			return alwaysFalse, fmt.Errorf("invalid constraint float value: %s", err)
		}
		propertyPredicate = func(propertyValue interface{}) (bool, error) {
			actualValue, ok := propertyValue.(float64)
			if !ok {
				return false, nil
			}
			return actualValue > targetValue, nil
		}
	case models.ConstraintOperatorGTE:
		targetValue, err := strconv.ParseFloat(strings.Trim(c.Value, " "), 64)
		if err != nil {
			return alwaysFalse, fmt.Errorf("invalid constraint float value: %s", err)
		}
		propertyPredicate = func(propertyValue interface{}) (bool, error) {
			actualValue, ok := propertyValue.(float64)
			if !ok {
				return false, nil
			}
			return actualValue >= targetValue, nil
		}
	case models.ConstraintOperatorEREG:
		regex, err := regexp.Compile(strings.Trim(c.Value, " \""))
		if err != nil {
			return alwaysFalse, fmt.Errorf("invalid constraint regex: %s", err)
		}
		propertyPredicate = func(propertyValue interface{}) (bool, error) {
			actualValue, ok := propertyValue.(string)
			if !ok {
				return false, nil
			}
			return regex.MatchString(actualValue), nil
		}
	case models.ConstraintOperatorNEREG:
		regex, err := regexp.Compile(strings.Trim(c.Value, " \""))
		if err != nil {
			return alwaysFalse, fmt.Errorf("invalid constraint regex: %s", err)
		}
		propertyPredicate = func(propertyValue interface{}) (bool, error) {
			actualValue, ok := propertyValue.(string)
			if !ok {
				return false, nil
			}
			return !regex.MatchString(actualValue), nil
		}
	case models.ConstraintOperatorIN:
		var targetValues []interface{}
		if err := json.Unmarshal([]byte(strings.Trim(c.Value, " ")), &targetValues); err != nil {
			return alwaysFalse, fmt.Errorf("invalid constraint array value: %s", err)
		}
		propertyPredicate = func(propertyValue interface{}) (bool, error) {
			for i := range targetValues {
				if reflect.DeepEqual(propertyValue, targetValues[i]) {
					return true, nil
				}
			}
			return false, nil
		}
	case models.ConstraintOperatorNOTIN:
		var targetValues []interface{}
		if err := json.Unmarshal([]byte(strings.Trim(c.Value, " ")), &targetValues); err != nil {
			return alwaysFalse, fmt.Errorf("invalid constraint array value: %s", err)
		}
		propertyPredicate = func(propertyValue interface{}) (bool, error) {
			for i := range targetValues {
				if reflect.DeepEqual(propertyValue, targetValues[i]) {
					return false, nil
				}
			}
			return true, nil
		}
	case models.ConstraintOperatorCONTAINS:
		var targetValue interface{}
		if err := json.Unmarshal([]byte(strings.Trim(c.Value, " ")), &targetValue); err != nil {
			return alwaysFalse, fmt.Errorf("invalid constraint value: %s", err)
		}
		propertyPredicate = func(propertyValue interface{}) (bool, error) {
			actualValues, ok := propertyValue.([]interface{})
			if !ok {
				return false, fmt.Errorf("%s entity_context value is not a slice: %v", c.Property, propertyValue)
			}
			for i := range actualValues {
				if reflect.DeepEqual(actualValues[i], targetValue) {
					return true, nil
				}
			}
			return false, nil
		}
	case models.ConstraintOperatorNOTCONTAINS:
		var targetValue interface{}
		if err := json.Unmarshal([]byte(strings.Trim(c.Value, " ")), &targetValue); err != nil {
			return alwaysFalse, fmt.Errorf("invalid constraint value: %s", err)
		}
		propertyPredicate = func(propertyValue interface{}) (bool, error) {
			actualValues, ok := propertyValue.([]interface{})
			if !ok {
				return false, fmt.Errorf("%s entity_context value is not a slice: %v", c.Property, propertyValue)
			}
			for i := range actualValues {
				if reflect.DeepEqual(actualValues[i], targetValue) {
					return false, nil
				}
			}
			return true, nil
		}
	}

	return func(entityContext map[string]interface{}) (bool, error) {
		propertyValue, exists := entityContext[c.Property]
		if !exists {
			return false, nil
		}
		return propertyPredicate(propertyValue)
	}, nil
}
