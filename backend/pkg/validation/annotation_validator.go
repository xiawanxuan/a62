package validation

import (
	"sonar-annotation-backend/internal/models"
)

type ValidationError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

func ValidateAnnotation(typ string, points []models.Point) []ValidationError {
	var errors []ValidationError

	if typ != "rectangle" && typ != "polygon" {
		errors = append(errors, ValidationError{
			Field:   "type",
			Message: "invalid annotation type, must be 'rectangle' or 'polygon'",
		})
	}

	if len(points) < 2 {
		errors = append(errors, ValidationError{
			Field:   "points",
			Message: "at least 2 points required",
		})
		return errors
	}

	if typ == "rectangle" {
		if len(points) != 2 {
			errors = append(errors, ValidationError{
				Field:   "points",
				Message: "rectangle requires exactly 2 points",
			})
		} else {
			if points[0].X == points[1].X || points[0].Y == points[1].Y {
				errors = append(errors, ValidationError{
					Field:   "points",
					Message: "rectangle points must form a valid rectangle",
				})
			}
		}
	}

	if typ == "polygon" {
		if len(points) < 3 {
			errors = append(errors, ValidationError{
				Field:   "points",
				Message: "polygon requires at least 3 points",
			})
		}

		if !isValidPolygon(points) {
			errors = append(errors, ValidationError{
				Field:   "points",
				Message: "invalid polygon geometry",
			})
		}
	}

	return errors
}

func isValidPolygon(points []models.Point) bool {
	n := len(points)
	if n < 3 {
		return false
	}

	for i := 0; i < n; i++ {
		if points[i].X < 0 || points[i].Y < 0 {
			return false
		}
	}

	return true
}

func ValidateFileSize(size int64, maxSize int64) []ValidationError {
	var errors []ValidationError
	if size <= 0 {
		errors = append(errors, ValidationError{
			Field:   "size",
			Message: "invalid file size",
		})
	}
	if size > maxSize {
		errors = append(errors, ValidationError{
			Field:   "size",
			Message: "file too large",
		})
	}
	return errors
}

func ValidateCategoryName(name string) []ValidationError {
	var errors []ValidationError
	if name == "" {
		errors = append(errors, ValidationError{
			Field:   "name",
			Message: "category name is required",
		})
	}
	if len(name) > 100 {
		errors = append(errors, ValidationError{
			Field:   "name",
			Message: "category name too long",
		})
	}
	return errors
}
