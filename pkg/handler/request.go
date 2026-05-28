package handler

import "github.com/labstack/echo/v4"

// bindRequest is a generic helper that binds the request body to any type T.
func bindRequest[T any](c echo.Context) (*T, error) {
	var req T
	if err := c.Bind(&req); err != nil {
		return nil, err
	}
	return &req, nil
}
