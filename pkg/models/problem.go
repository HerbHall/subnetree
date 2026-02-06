package models

// APIProblem represents an RFC 7807 Problem Details response for Swagger docs.
// This type is used only in swagger annotations to describe error responses.
type APIProblem struct {
	Type     string `json:"type" example:"https://subnetree.com/problems/bad-request"`
	Title    string `json:"title" example:"Bad Request"`
	Status   int    `json:"status" example:"400"`
	Detail   string `json:"detail,omitempty" example:"invalid request body"`
	Instance string `json:"instance,omitempty" example:"/api/v1/auth/login"`
}
