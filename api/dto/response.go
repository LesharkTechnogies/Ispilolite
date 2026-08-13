package dto

// APIResponse is the standard success/error envelope used across the API.
type APIResponse struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data"`
	Message string      `json:"message,omitempty"`
	Errors  interface{} `json:"errors,omitempty"`
}

// SearchMeta describes how a search result set was produced. It lets clients
// (and tests) know whether the fast path (elasticsearch) or the fallback
// (postgres) served the request.
type SearchMeta struct {
	Source    string `json:"source"`     // "elasticsearch" | "postgres_fallback"
	Total     int    `json:"total"`      // total hits (may exceed returned page)
	Page      int    `json:"page"`       // 1-based page number
	PageSize  int    `json:"page_size"`  // items per page
	TookMS    int64  `json:"took_ms"`    // server-side latency in milliseconds
	Query     string `json:"query,omitempty"`
	Fallback  bool   `json:"fallback"`   // true when ES was unavailable/failed
	Degraded  bool   `json:"degraded,omitempty"` // true when results are approximate
}

// Suggestion is a single "did you mean" / autocomplete candidate.
type Suggestion struct {
	Text  string  `json:"text"`
	Type  string  `json:"type,omitempty"`  // county, village, isp, technician ...
	Score float64 `json:"score,omitempty"`
}

// SearchResult is the generic body returned by every search endpoint.
type SearchResult struct {
	Items       interface{}  `json:"items"`
	Suggestions []Suggestion `json:"suggestions,omitempty"`
	DidYouMean  string       `json:"did_you_mean,omitempty"`
	Meta        SearchMeta   `json:"meta"`
}

type TokenResponse struct {
    AccessToken  string `json:"access_token"`
    RefreshToken string `json:"refresh_token"`
    TokenType    string `json:"token_type"`
    ExpiresIn    int    `json:"expires_in"`
}

type UserProfileResponse struct {
    ID             string                `json:"id"`
    Name           string                `json:"name"`
    Phone          string                `json:"phone"`
    Email          string                `json:"email"`
    Role           string                `json:"role"`
    IsVerified     bool                  `json:"is_verified"`
    Rating         float64               `json:"rating"`
    TotalRatings   int                   `json:"total_ratings"`
    Joined         string                `json:"joined"`
    Location       *LocationDTO          `json:"location,omitempty"`
    Statistics     *UserStatisticsDTO    `json:"statistics,omitempty"`
}

type UserStatisticsDTO struct {
    CompletedInstallations int `json:"completed_installations"`
    PendingRequests        int `json:"pending_requests"`
    FavoriteISPs           int `json:"favorite_isps"`
    FavoriteTechnicians    int `json:"favorite_technicians"`
}

type ISPProfileResponse struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Description string   `json:"description,omitempty"`
	AvatarURL   string   `json:"avatar_url,omitempty"`
	County      string   `json:"county,omitempty"`
	SubCounty   string   `json:"sub_county,omitempty"`
	Villages    []string `json:"villages,omitempty"`
	Rating      float64  `json:"rating"`
	ReviewCount int      `json:"review_count"`
	IsActive    bool     `json:"is_active"`
}

type ISPStatisticsDTO struct {
    CompletedInstallations int `json:"completed_installations"`
    PendingRequests        int `json:"pending_requests"`
    TechniciansCount       int `json:"technicians_count"`
}

type TechProfileResponse struct {
    ID             string                `json:"id"`
    Name           string                `json:"name"`
    Phone          string                `json:"phone"`
    Email          string                `json:"email"`
    Role           string                `json:"role"`
    IsVerified     bool                  `json:"is_verified"`
    Rating         float64               `json:"rating"`
    TotalRatings   int                   `json:"total_ratings"`
    Joined         string                `json:"joined"`
    Location       *LocationDTO          `json:"location,omitempty"`
    Statistics     *TechStatisticsDTO    `json:"statistics,omitempty"`
}

type TechStatisticsDTO struct {
    CompletedJobs int `json:"completed_jobs"`
    PendingJobs   int `json:"pending_jobs"`
}
