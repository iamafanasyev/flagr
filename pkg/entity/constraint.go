package entity

import (
	"encoding/json"
	"fmt"
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

func (cs ConstraintArray) Match(entityContext map[string]interface{}) (bool, error) {
	for i := range cs {
		match, err := cs[i].Match(entityContext)
		if err != nil || !match {
			return false, err
		}
	}
	return true, nil
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
		if err := json.Unmarshal([]byte(strings.TrimSpace(c.Value)), &targetValue); err != nil {
			return alwaysFalse, fmt.Errorf("invalid eq constraint target value: %s", err)
		}
		switch targetValue.(type) {
		case float64:
			propertyPredicate = func(propertyValue interface{}) (bool, error) {
				actualValue, err := AsFloat64(propertyValue)
				if err != nil {
					return false, nil
				}
				return actualValue == targetValue, nil
			}
			break
		case string:
			propertyPredicate = func(propertyValue interface{}) (bool, error) {
				actualValue, ok := propertyValue.(string)
				if !ok {
					return false, nil
				}
				return actualValue == targetValue, nil
			}
			break
		case bool:
			propertyPredicate = func(propertyValue interface{}) (bool, error) {
				actualValue, ok := propertyValue.(bool)
				if !ok {
					return false, nil
				}
				return actualValue == targetValue, nil
			}
			break
		default:
			return alwaysFalse, fmt.Errorf("unsupported eq constraint target value type: %T", targetValue)
		}
	case models.ConstraintOperatorNEQ:
		var targetValue interface{}
		if err := json.Unmarshal([]byte(strings.TrimSpace(c.Value)), &targetValue); err != nil {
			return alwaysFalse, fmt.Errorf("invalid eq constraint target value: %s", err)
		}
		switch targetValue.(type) {
		case float64:
			propertyPredicate = func(propertyValue interface{}) (bool, error) {
				actualValue, err := AsFloat64(propertyValue)
				if err != nil {
					return true, nil
				}
				return actualValue != targetValue, nil
			}
			break
		case string:
			propertyPredicate = func(propertyValue interface{}) (bool, error) {
				actualValue, ok := propertyValue.(string)
				if !ok {
					return true, nil
				}
				return actualValue != targetValue, nil
			}
			break
		case bool:
			propertyPredicate = func(propertyValue interface{}) (bool, error) {
				actualValue, ok := propertyValue.(bool)
				if !ok {
					return true, nil
				}
				return actualValue != targetValue, nil
			}
			break
		default:
			return alwaysFalse, fmt.Errorf("unsupported eq constraint target value type: %T", targetValue)
		}
	case models.ConstraintOperatorLT:
		targetValue, err := strconv.ParseFloat(strings.TrimSpace(c.Value), 64)
		if err != nil {
			return alwaysFalse, fmt.Errorf("invalid lt constraint target value: %s", err)
		}
		propertyPredicate = func(propertyValue interface{}) (bool, error) {
			actualValue, err := AsFloat64(propertyValue)
			if err != nil {
				return false, nil
			}
			return actualValue < targetValue, nil
		}
	case models.ConstraintOperatorLTE:
		targetValue, err := strconv.ParseFloat(strings.TrimSpace(c.Value), 64)
		if err != nil {
			return alwaysFalse, fmt.Errorf("invalid lte constraint target value: %s", err)
		}
		propertyPredicate = func(propertyValue interface{}) (bool, error) {
			actualValue, err := AsFloat64(propertyValue)
			if err != nil {
				return false, nil
			}
			return actualValue <= targetValue, nil
		}
	case models.ConstraintOperatorGT:
		targetValue, err := strconv.ParseFloat(strings.TrimSpace(c.Value), 64)
		if err != nil {
			return alwaysFalse, fmt.Errorf("invalid gt constraint target value: %s", err)
		}
		propertyPredicate = func(propertyValue interface{}) (bool, error) {
			actualValue, err := AsFloat64(propertyValue)
			if err != nil {
				return false, nil
			}
			return actualValue > targetValue, nil
		}
	case models.ConstraintOperatorGTE:
		targetValue, err := strconv.ParseFloat(strings.TrimSpace(c.Value), 64)
		if err != nil {
			return alwaysFalse, fmt.Errorf("invalid gte constraint target value: %s", err)
		}
		propertyPredicate = func(propertyValue interface{}) (bool, error) {
			actualValue, err := AsFloat64(propertyValue)
			if err != nil {
				return false, nil
			}
			return actualValue >= targetValue, nil
		}
	case models.ConstraintOperatorEREG:
		var regexString string
		if err := json.Unmarshal([]byte(strings.TrimSpace(c.Value)), &regexString); err != nil {
			return alwaysFalse, fmt.Errorf("%s ereg constraint value should be escaped regex: %s", c.Property, err)
		}
		regex, err := regexp.Compile(regexString)
		if err != nil {
			return alwaysFalse, fmt.Errorf("invalid ereg constraint regex: %s", err)
		}
		propertyPredicate = func(propertyValue interface{}) (bool, error) {
			actualValue, ok := propertyValue.(string)
			if !ok {
				return false, nil
			}
			return regex.MatchString(actualValue), nil
		}
	case models.ConstraintOperatorNEREG:
		var regexString string
		if err := json.Unmarshal([]byte(strings.TrimSpace(c.Value)), &regexString); err != nil {
			return alwaysFalse, fmt.Errorf("%s nereg constraint value should be escaped regex: %s", c.Property, err)
		}
		regex, err := regexp.Compile(regexString)
		if err != nil {
			return alwaysFalse, fmt.Errorf("invalid nereg constraint regex: %s", err)
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
		if err := json.Unmarshal([]byte(strings.TrimSpace(c.Value)), &targetValues); err != nil {
			return alwaysFalse, fmt.Errorf("invalid in constraint target value: %s", err)
		}
		propertyPredicate = func(propertyValue interface{}) (bool, error) {
			return false, nil
		}
		for i := range targetValues {
			switch targetValues[i].(type) {
			case float64:
				propertyPredicate = func(propertyValue interface{}) (bool, error) {
					actualValue, err := AsFloat64(propertyValue)
					if err != nil {
						return false, nil
					}
					for j := range targetValues {
						if targetValues[j] == actualValue {
							return true, nil
						}
					}
					return false, nil
				}
				break
			case string:
				propertyPredicate = func(propertyValue interface{}) (bool, error) {
					actualValue, ok := propertyValue.(string)
					if !ok {
						return false, nil
					}
					for j := range targetValues {
						if targetValues[j] == actualValue {
							return true, nil
						}
					}
					return false, nil
				}
				break
			default:
				return alwaysFalse, fmt.Errorf("unsupported in constraint target value type: %T", targetValues[i])
			}
		}
	case models.ConstraintOperatorNOTIN:
		var targetValues []interface{}
		if err := json.Unmarshal([]byte(strings.TrimSpace(c.Value)), &targetValues); err != nil {
			return alwaysFalse, fmt.Errorf("invalid notin constraint target value: %s", err)
		}
		propertyPredicate = func(propertyValue interface{}) (bool, error) {
			return true, nil
		}
		for i := range targetValues {
			switch targetValues[i].(type) {
			case float64:
				propertyPredicate = func(propertyValue interface{}) (bool, error) {
					actualValue, err := AsFloat64(propertyValue)
					if err != nil {
						return true, nil
					}
					for j := range targetValues {
						if targetValues[j] == actualValue {
							return false, nil
						}
					}
					return true, nil
				}
				break
			case string:
				propertyPredicate = func(propertyValue interface{}) (bool, error) {
					actualValue, ok := propertyValue.(string)
					if !ok {
						return true, nil
					}
					for j := range targetValues {
						if targetValues[j] == actualValue {
							return false, nil
						}
					}
					return true, nil
				}
				break
			default:
				return alwaysFalse, fmt.Errorf("unsupported notin constraint target value type: %T", targetValues[i])
			}
		}
	case models.ConstraintOperatorCONTAINS:
		var targetValue interface{}
		if err := json.Unmarshal([]byte(strings.TrimSpace(c.Value)), &targetValue); err != nil {
			return alwaysFalse, fmt.Errorf("invalid contains constraint value: %s", err)
		}
		propertyPredicate = func(propertyValue interface{}) (bool, error) {
			return false, nil
		}
		switch targetValue.(type) {
		case float64:
			propertyPredicate = func(propertyValue interface{}) (bool, error) {
				actualValues, ok := propertyValue.([]interface{})
				if !ok {
					return false, nil
				}
				for i := range actualValues {
					if actualValue, err := AsFloat64(actualValues[i]); err == nil {
						if actualValue == targetValue {
							return true, nil
						}
					}
				}
				return false, nil
			}
			break
		case string:
			propertyPredicate = func(propertyValue interface{}) (bool, error) {
				actualValues, ok := propertyValue.([]interface{})
				if !ok {
					return false, nil
				}
				for i := range actualValues {
					if actualValue, ok := actualValues[i].(string); ok {
						if actualValue == targetValue {
							return true, nil
						}
					}
				}
				return false, nil
			}
			break
		default:
			return alwaysFalse, fmt.Errorf("unsupported contains constraint target value type: %T", targetValue)
		}
	case models.ConstraintOperatorNOTCONTAINS:
		var targetValue interface{}
		if err := json.Unmarshal([]byte(strings.TrimSpace(c.Value)), &targetValue); err != nil {
			return alwaysFalse, fmt.Errorf("invalid notcontains constraint value: %s", err)
		}
		propertyPredicate = func(propertyValue interface{}) (bool, error) {
			return true, nil
		}
		switch targetValue.(type) {
		case float64:
			propertyPredicate = func(propertyValue interface{}) (bool, error) {
				actualValues, ok := propertyValue.([]interface{})
				if !ok {
					return true, nil
				}
				for i := range actualValues {
					if actualValue, err := AsFloat64(actualValues[i]); err == nil {
						if actualValue == targetValue {
							return false, nil
						}
					}
				}
				return true, nil
			}
			break
		case string:
			propertyPredicate = func(propertyValue interface{}) (bool, error) {
				actualValues, ok := propertyValue.([]interface{})
				if !ok {
					return true, nil
				}
				for i := range actualValues {
					if actualValue, ok := actualValues[i].(string); ok {
						if actualValue == targetValue {
							return false, nil
						}
					}
				}
				return true, nil
			}
			break
		default:
			return alwaysFalse, fmt.Errorf("unsupported notcontains constraint target value type: %T", targetValue)
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

func AsFloat64(v interface{}) (float64, error) {
	switch v.(type) {
	case int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64:
		return float64(v.(int)), nil
	case float32, float64:
		return v.(float64), nil
	}
	return 0, fmt.Errorf("value is not a numeric: %v", v)
}
